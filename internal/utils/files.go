package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/gabriel-vasile/mimetype"
)

// SampleHashFile calculates a hash based on small file segments
func SampleHashFile(path string, threads int, gap float64, chunkSize int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", err
	}

	size := info.Size()
	if size == 0 {
		return "", nil
	}

	if chunkSize <= 0 {
		// Linear interpolation for chunk size based on file size
		dataPoints := [][2]float64{
			{26214400, 262144},      // 25MB -> 256KB
			{52428800000, 10485760}, // 50GB -> 10MB
		}
		chunkSize = int64(LinearInterpolation(float64(size), dataPoints))
	}

	segments := CalculateSegmentsInt(size, chunkSize, gap)
	if len(segments) == 0 {
		return "", nil
	}

	hashes := make([][]byte, len(segments))
	var wg sync.WaitGroup

	if threads <= 0 {
		threads = 1
	}

	sem := make(chan struct{}, threads)

	for i, start := range segments {
		wg.Add(1)
		go func(idx int, offset int64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			buf := make([]byte, chunkSize)
			n, err := file.ReadAt(buf, offset)
			if err != nil && err != io.EOF {
				slog.Error("Read error during hashing", "path", path, "offset", offset, "error", err)
				return
			}
			data := buf[:n]
			h := sha256.New()
			h.Write(data)
			hashes[idx] = h.Sum(nil)
		}(i, start)
	}

	wg.Wait()

	// Final hash of all segment hashes
	finalHash := sha256.New()
	for _, h := range hashes {
		if h != nil {
			finalHash.Write(h)
		}
	}

	return fmt.Sprintf("%x", finalHash.Sum(nil)), nil
}

// FullHashFile calculates a full sha256 hash of a file
func FullHashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// FilterDeleted returns only the paths that currently exist on the filesystem
func FilterDeleted(paths []string) []string {
	var existing []string
	deletedDirs := make(map[string]bool)

	for _, p := range paths {
		dir := filepath.Dir(p)
		if deletedDirs[dir] {
			continue
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			deletedDirs[dir] = true
			continue
		}

		if _, err := os.Stat(p); err == nil {
			existing = append(existing, p)
		}
	}
	return existing
}

type FileStats struct {
	Size         int64
	TimeCreated  int64
	TimeModified int64
}

// GetFileStats returns size and timestamps for a file
func GetFileStats(path string) (FileStats, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return FileStats{}, err
	}

	return FileStats{
		Size:         stat.Size(),
		TimeCreated:  stat.ModTime().Unix(), // Go doesn't have a cross-platform way to get creation time easily
		TimeModified: stat.ModTime().Unix(),
	}, nil
}

// IsFileOpen checks if a file is currently open by any process
func IsFileOpen(path string) bool {
	if runtime.GOOS == "windows" {
		// On Windows, try to open the file with exclusive access
		f, err := os.OpenFile(path, os.O_RDWR, 0)
		if err != nil {
			return true
		}
		f.Close()
		return false
	}

	if runtime.GOOS == "linux" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}

		files, err := os.ReadDir("/proc")
		if err != nil {
			return false
		}

		for _, f := range files {
			if !f.IsDir() {
				continue
			}
			// Check if name is a number (PID)
			isPid := true
			for _, r := range f.Name() {
				if r < '0' || r > '9' {
					isPid = false
					break
				}
			}
			if !isPid {
				continue
			}

			fdDir := filepath.Join("/proc", f.Name(), "fd")
			fds, err := os.ReadDir(fdDir)
			if err != nil {
				continue
			}

			for _, fd := range fds {
				link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
				if err == nil && link == absPath {
					return true
				}
			}
		}
	}

	return false
}

// DetectMimeType returns the mimetype of a file
func DetectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".apk" {
		return "application/vnd.android.package-archive"
	}
	if ext == ".zim" {
		return "application/x-zim"
	}
	if ext == ".epub" {
		return "application/epub+zip"
	}
	if ext == ".pdf" {
		return "application/pdf"
	}
	if ext == ".mobi" {
		return "application/x-mobipocket-ebook"
	}

	mime, err := mimetype.DetectFile(path)
	if err != nil {
		return ""
	}
	return mime.String()
}

// Rename renames a file, respecting simulation mode
func Rename(flags models.GlobalFlags, src, dst string) error {
	if flags.Simulate {
		fmt.Printf("rename %s %s\n", src, dst)
		return nil
	}
	slog.Debug("rename", "src", src, "dst", dst)
	return os.Rename(src, dst)
}

// Unlink deletes a file, respecting simulation mode
func Unlink(flags models.GlobalFlags, path string) error {
	if flags.Simulate {
		fmt.Printf("unlink %s\n", path)
		return nil
	}
	slog.Debug("unlink", "path", path)
	return os.Remove(path)
}

// Rmtree deletes a directory tree, respecting simulation mode
func Rmtree(flags models.GlobalFlags, path string) error {
	if flags.Simulate {
		fmt.Printf("rmtree %s\n", path)
		return nil
	}
	slog.Debug("rmtree", "path", path)
	return os.RemoveAll(path)
}

// AltName returns an alternative filename if the given path already exists
func AltName(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	counter := 1
	for {
		newPath := fmt.Sprintf("%s_%d%s", base, counter, ext)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
		counter++
	}
}

// CommonPath returns the longest common path prefix
func CommonPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	if len(paths) == 1 {
		return filepath.Dir(paths[0])
	}

	sep := string(filepath.Separator)
	parts := strings.Split(filepath.Clean(paths[0]), sep)

	for i := 1; i < len(paths); i++ {
		p := strings.Split(filepath.Clean(paths[i]), sep)
		if len(p) < len(parts) {
			parts = parts[:len(p)]
		}
		for j := 0; j < len(parts); j++ {
			if parts[j] != p[j] {
				parts = parts[:j]
				break
			}
		}
	}

	if len(parts) == 0 {
		return sep
	}
	return strings.Join(parts, sep)
}

// CommonPathFull returns a common path prefix.
// Previously it included common words in the suffix, but this was confusing for UI.
func CommonPathFull(paths []string) string {
	return CommonPath(paths)
}

// GetExternalSubtitles finds external subtitle files associated with a media file
// Supports patterns: movie.srt, movie.en.srt, movie_eng.srt, movie.EN.srt, movie.eng.srt, etc.
func GetExternalSubtitles(path string) []string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)

	var subs []string
	subExts := []string{".srt", ".vtt", ".ass", ".ssa", ".lrc", ".idx", ".sub"}

	for _, sExt := range subExts {
		// Exact match: movie.srt
		subPath := base + sExt
		if FileExists(subPath) {
			subs = append(subs, subPath)
		}

		// Pattern: movie.<lang>.srt (e.g., movie.en.srt, movie.eng.srt)
		matches, _ := filepath.Glob(base + ".*" + sExt)
		for _, m := range matches {
			if !strings.EqualFold(m, subPath) {
				subs = append(subs, m)
			}
		}

		// Pattern: movie_<lang>.srt (e.g., movie_en.srt, movie_eng.srt)
		matches2, _ := filepath.Glob(base + "_*" + sExt)
		subs = append(subs, matches2...)

		// Pattern: movie - <lang>.srt (e.g., movie - English.srt)
		matches3, _ := filepath.Glob(base + " - *" + sExt)
		subs = append(subs, matches3...)
	}

	return Unique(subs)
}

// ExtractSubtitleInfo extracts language and codec information from a subtitle filename
// Returns (displayName, languageCode, codec)
// Examples:
//   - movie.en.srt -> "English (ssa)", "en", "ssa" (ass is displayed as ssa)
//   - movie_eng.ass -> "English (ssa)", "eng", "ssa"
//   - movie.srt -> "srt", "", "srt"
//   - movie.EN.srt -> "English (ssa)", "en", "ssa"
//   - movie - English.srt -> "English (srt)", "en", "srt" (full language names supported)
func ExtractSubtitleInfo(subPath string) (displayName, languageCode, codec string) {
	base := filepath.Base(subPath)
	ext := strings.ToLower(filepath.Ext(base))
	codec = strings.TrimPrefix(ext, ".")

	// Always display 'ass' as 'ssa'
	if codec == "ass" {
		codec = "ssa"
	}

	if codec == "" {
		return "", "", ""
	}

	// Remove the extension to get the base name
	nameWithoutExt := strings.TrimSuffix(base, ext)

	// Try to extract language code from patterns:
	// movie.en.srt, movie_eng.srt, movie.EN.srt, movie.eng.srt, movie - English.srt, etc.

	// Pattern 1: movie.<lang>.ext or movie_<lang>.ext
	// Look for the last separator (dot or underscore) before the extension
	// Check dot separator first (most common)
	dotIdx := strings.LastIndex(nameWithoutExt, ".")
	underscoreIdx := strings.LastIndex(nameWithoutExt, "_")

	var langCode string

	// Try dot first
	if dotIdx != -1 {
		potentialLang := nameWithoutExt[dotIdx+1:]
		if isLanguageCode(potentialLang) {
			langCode = strings.ToLower(potentialLang)
		}
	}

	// If not found with dot, try underscore
	if langCode == "" && underscoreIdx != -1 {
		potentialLang := nameWithoutExt[underscoreIdx+1:]
		if isLanguageCode(potentialLang) {
			langCode = strings.ToLower(potentialLang)
		}
	}

	// Try to parse full language names from dash notation (e.g., "movie - English.srt")
	if langCode == "" {
		dashIdx := strings.LastIndex(nameWithoutExt, " - ")
		if dashIdx != -1 {
			potentialLang := nameWithoutExt[dashIdx+3:]
			// Try to match full language name directly
			code := getLanguageCode(potentialLang)
			if code != "" {
				langCode = code
			}
		}
	}

	if langCode != "" {
		langName := getLanguageName(langCode)
		if langName != "" {
			return langName + " (" + codec + ")", langCode, codec
		}
		return langCode + " (" + codec + ")", langCode, codec
	}

	// No language detected, return codec in parentheses to match frontend's external sub regex
	return "(" + codec + ")", "", codec
}

// isLanguageCode checks if a string looks like a language code
func isLanguageCode(s string) bool {
	if len(s) < 2 || len(s) > 4 {
		return false
	}

	// Common 2-letter and 3-letter language codes
	validCodes := map[string]bool{
		// 2-letter codes
		"en": true, "es": true, "fr": true, "de": true, "it": true, "pt": true,
		"ru": true, "ja": true, "ko": true, "zh": true, "ar": true, "hi": true,
		"nl": true, "pl": true, "tr": true, "sv": true, "no": true, "da": true,
		"fi": true, "el": true, "he": true, "th": true, "vi": true, "id": true,
		"ms": true, "tl": true, "uk": true, "cs": true, "sk": true, "hu": true,
		"ro": true, "bg": true, "hr": true, "sr": true, "sl": true, "et": true,
		"lv": true, "lt": true, "fa": true, "ur": true, "bn": true, "ta": true,
		"te": true, "mr": true, "gu": true, "kn": true, "ml": true, "pa": true,
		"or": true, "my": true, "km": true, "lo": true, "ka": true, "am": true,
		"sw": true, "zu": true, "xh": true, "af": true, "sq": true, "az": true,
		"be": true, "bs": true, "ca": true, "eu": true, "gl": true, "is": true,
		"ga": true, "mk": true, "mn": true, "ne": true, "si": true, "uz": true,
		"kk": true, "hy": true, "ps": true, "sd": true, "tk": true, "tg": true,
		"ky": true, "so": true, "yo": true, "ig": true, "ha": true,
		// 3-letter codes
		"eng": true, "spa": true, "fra": true, "deu": true, "ita": true, "por": true,
		"rus": true, "jpn": true, "kor": true, "zho": true, "ara": true, "hin": true,
		"nld": true, "pol": true, "tur": true, "swe": true, "nor": true, "dan": true,
		"fin": true, "ell": true, "heb": true, "tha": true, "vie": true, "ind": true,
		"msa": true, "fil": true, "ukr": true, "ces": true, "slk": true, "hun": true,
		"ron": true, "bul": true, "hrv": true, "srp": true, "slv": true, "est": true,
		"lav": true, "lit": true, "fas": true, "urd": true, "ben": true, "tam": true,
		"tel": true, "mar": true, "guj": true, "kan": true, "mal": true, "pan": true,
		"ori": true, "bur": true, "khm": true, "lao": true, "geo": true, "amh": true,
	}

	return validCodes[strings.ToLower(s)]
}

// getLanguageName converts a language code to its full name
func getLanguageName(code string) string {
	code = strings.ToLower(code)

	// 2-letter to name mapping
	twoLetter := map[string]string{
		"en": "English", "es": "Spanish", "fr": "French", "de": "German",
		"it": "Italian", "pt": "Portuguese", "ru": "Russian", "ja": "Japanese",
		"ko": "Korean", "zh": "Chinese", "ar": "Arabic", "hi": "Hindi",
		"nl": "Dutch", "pl": "Polish", "tr": "Turkish", "sv": "Swedish",
		"no": "Norwegian", "da": "Danish", "fi": "Finnish", "el": "Greek",
		"he": "Hebrew", "th": "Thai", "vi": "Vietnamese", "id": "Indonesian",
		"ms": "Malay", "tl": "Filipino", "uk": "Ukrainian", "cs": "Czech",
		"sk": "Slovak", "hu": "Hungarian", "ro": "Romanian", "bg": "Bulgarian",
		"hr": "Croatian", "sr": "Serbian", "sl": "Slovenian", "et": "Estonian",
		"lv": "Latvian", "lt": "Lithuanian", "fa": "Persian", "ur": "Urdu",
		"bn": "Bengali", "ta": "Tamil", "te": "Telugu", "mr": "Marathi",
		"gu": "Gujarati", "kn": "Kannada", "ml": "Malayalam", "pa": "Punjabi",
		"or": "Odia", "my": "Burmese", "km": "Khmer", "lo": "Lao",
		"ka": "Georgian", "am": "Amharic", "sw": "Swahili", "zu": "Zulu",
		"xh": "Xhosa", "af": "Afrikaans", "sq": "Albanian", "az": "Azerbaijani",
		"be": "Belarusian", "bs": "Bosnian", "ca": "Catalan", "eu": "Basque",
		"gl": "Galician", "is": "Icelandic", "ga": "Irish", "mk": "Macedonian",
		"mn": "Mongolian", "ne": "Nepali", "si": "Sinhala", "uz": "Uzbek",
		"kk": "Kazakh", "hy": "Armenian", "ps": "Pashto", "sd": "Sindhi",
		"tk": "Turkmen", "tg": "Tajik", "ky": "Kyrgyz", "so": "Somali",
		"yo": "Yoruba", "ig": "Igbo", "ha": "Hausa",
	}

	// 3-letter to name mapping
	threeLetter := map[string]string{
		"eng": "English", "spa": "Spanish", "fra": "French", "deu": "German",
		"ita": "Italian", "por": "Portuguese", "rus": "Russian", "jpn": "Japanese",
		"kor": "Korean", "zho": "Chinese", "ara": "Arabic", "hin": "Hindi",
		"nld": "Dutch", "pol": "Polish", "tur": "Turkish", "swe": "Swedish",
		"nor": "Norwegian", "dan": "Danish", "fin": "Finnish", "ell": "Greek",
		"heb": "Hebrew", "tha": "Thai", "vie": "Vietnamese", "ind": "Indonesian",
		"msa": "Malay", "fil": "Filipino", "ukr": "Ukrainian", "ces": "Czech",
		"slk": "Slovak", "hun": "Hungarian", "ron": "Romanian", "bul": "Bulgarian",
		"hrv": "Croatian", "srp": "Serbian", "slv": "Slovenian", "est": "Estonian",
		"lav": "Latvian", "lit": "Lithuanian", "fas": "Persian", "urd": "Urdu",
		"ben": "Bengali", "tam": "Tamil", "tel": "Telugu", "mar": "Marathi",
		"guj": "Gujarati", "kan": "Kannada", "mal": "Malayalam", "pan": "Punjabi",
		"ori": "Odia", "bur": "Burmese", "khm": "Khmer", "lao": "Lao",
		"geo": "Georgian", "amh": "Amharic",
	}

	if name, ok := threeLetter[code]; ok {
		return name
	}
	if name, ok := twoLetter[code]; ok {
		return name
	}

	return ""
}

// getLanguageCode converts a full language name to its code
func getLanguageCode(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))

	// Name to 2-letter code mapping (includes native names)
	nameToTwoLetter := map[string]string{
		// English names
		"english": "en", "spanish": "es", "french": "fr", "german": "de",
		"italian": "it", "portuguese": "pt", "russian": "ru", "japanese": "ja",
		"korean": "ko", "chinese": "zh", "arabic": "ar", "hindi": "hi",
		"dutch": "nl", "polish": "pl", "turkish": "tr", "swedish": "sv",
		"norwegian": "no", "danish": "da", "finnish": "fi", "greek": "el",
		"hebrew": "he", "thai": "th", "vietnamese": "vi", "indonesian": "id",
		"malay": "ms", "filipino": "tl", "ukrainian": "uk", "czech": "cs",
		"slovak": "sk", "hungarian": "hu", "romanian": "ro", "bulgarian": "bg",
		"croatian": "hr", "serbian": "sr", "slovenian": "sl", "estonian": "et",
		"latvian": "lv", "lithuanian": "lt", "persian": "fa", "urdu": "ur",
		"bengali": "bn", "tamil": "ta", "telugu": "te", "marathi": "mr",
		"gujarati": "gu", "kannada": "kn", "malayalam": "ml", "punjabi": "pa",
		"odia": "or", "burmese": "my", "khmer": "km", "lao": "lo",
		"georgian": "ka", "amharic": "am", "swahili": "sw", "zulu": "zu",
		"xhosa": "xh", "afrikaans": "af", "albanian": "sq", "azerbaijani": "az",
		"belarusian": "be", "bosnian": "bs", "catalan": "ca", "basque": "eu",
		"galician": "gl", "icelandic": "is", "irish": "ga", "macedonian": "mk",
		"mongolian": "mn", "nepali": "ne", "sinhala": "si", "uzbek": "uz",
		"kazakh": "kk", "armenian": "hy", "pashto": "ps", "sindhi": "sd",
		"turkmen": "tk", "tajik": "tg", "kyrgyz": "ky", "somali": "so",
		"yoruba": "yo", "igbo": "ig", "hausa": "ha",
		// Native names (non-English only)
		"deutsch": "de", "español": "es", "français": "fr", "italiano": "it",
		"português": "pt", "portugues": "pt", "русский": "ru", "russkij": "ru",
		"日本語": "ja", "한국어": "ko", "hangugeo": "ko", "中文": "zh",
		"العربية": "ar", "arabi": "ar", "हिन्दी": "hi", "nederlands": "nl",
		"polski": "pl", "türkçe": "tr", "turkce": "tr", "svenska": "sv",
		"norsk": "no", "dansk": "da", "suomi": "fi", "ελληνικά": "el",
		"ellinika": "el", "עברית": "he", "ivrit": "he", "ไทย": "th",
		"tiếng việt": "vi", "tieng viet": "vi", "bahasa indonesia": "id",
		"українська": "uk", "ukrainska": "uk", "čeština": "cs", "ceska": "cs",
		"slovenčina": "sk", "slovencina": "sk", "magyar": "hu", "română": "ro",
		"romana": "ro", "български": "bg", "bulgarski": "bg", "hrvatski": "hr",
		"српски": "sr", "srpski": "sr", "slovenščina": "sl", "slovenscina": "sl",
		"eesti": "et", "latviešu": "lv", "latviesu": "lv", "lietuvių": "lt",
		"lietuviu": "lt", "فارسی": "fa", "farsi": "fa", "اردو": "ur",
		"বাংলা": "bn", "bangla": "bn", "தமிழ்": "ta", "తెలుగు": "te",
		"मराठी": "mr", "ગુજરાતી": "gu", "ಕನ್ನಡ": "kn", "മലയാളം": "ml",
		"ਪੰਜਾਬੀ": "pa", "ଓଡ଼ିଆ": "or", "မြန်မာ": "my", "myanmar": "my",
		"ខ្មែរ": "km", "ລາວ": "lo", "ქართული": "ka", "kartuli": "ka",
		"አማርኛ": "am", "kiswahili": "sw", "isiZulu": "zu", "isiXhosa": "xh",
		"azərbaycan": "az", "беларуская": "be", "bosanski": "bs",
		"català": "ca", "catala": "ca", "euskara": "eu",
		"galego": "gl", "íslenska": "is", "islenska": "is", "gaeilge": "ga",
		"македонски": "mk", "makedonski": "mk", "монгол": "mn", "mongol": "mn",
		"नेपाली": "ne", "සිංහල": "si", "o'zbek": "uz", "ozbek": "uz",
		"қазақ": "kk", "հայերեն": "hy", "hayeren": "hy",
		"پښتو": "ps", "سنڌي": "sd", "türkmen": "tk", "тоҷикӣ": "tg", "tojiki": "tg",
		"кыргызча": "ky", "kyrgyzcha": "ky", "soomaali": "so", "yorùbá": "yo",
		"háusa": "ha",
	}

	if code, ok := nameToTwoLetter[name]; ok {
		return code
	}

	return ""
}
