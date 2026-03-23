package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	DefaultTableLimit        = 350
	DefaultPlayQueue         = 120
	DefaultSubtitleMix       = 0.35
	DefaultFileRowsReadLimit = 500000
	DefaultMultiplePlayback  = -1
	DefaultOpenLimit         = 7
)

func GetMpvListenSocket() string {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}
	return filepath.Join(runtimeDir, "mpv_socket")
}

func GetMpvWatchSocket() string {
	home, _ := os.UserHomeDir()
	if IsWindows {
		return filepath.Join(home, "AppData", "Roaming", "mpv", "socket")
	}
	if IsMac {
		return filepath.Join(home, "Library", "Application Support", "mpv", "socket")
	}
	return filepath.Join(home, ".config", "mpv", "socket")
}

func GetMpvWatchLaterDir() string {
	home, _ := os.UserHomeDir()
	if IsWindows {
		return filepath.Join(home, "AppData", "Roaming", "mpv", "watch_later")
	}
	if IsMac {
		return filepath.Join(home, "Library", "Application Support", "mpv", "watch_later")
	}
	return filepath.Join(home, ".config", "mpv", "watch_later")
}

var (
	ApplicationStart = time.Now().Unix()
	IsWindows        = runtime.GOOS == "windows"
	IsLinux          = runtime.GOOS == "linux"
	IsMac            = runtime.GOOS == "darwin"
	TERMINAL_SIZE    = struct{ columns, rows int }{80, 24}
)

var SQLiteExtensions = []string{".sqlite", ".sqlite3", ".db", ".db3", ".s3db", ".sl3"}

var AudioExtensions = []string{
	"aa3", "aac", "ac3", "aif", "aiff", "ape", "caf", "dff", "dsf", "flac",
	"m2a", "m4a", "m4b", "m4r", "mka", "mp3", "mpga", "oga", "ogg", "opus",
	"pcm", "wav", "wma",
}

var VideoExtensions = []string{
	"264", "265", "302", "3g2", "3gp", "3gp2", "3gpp", "669", "722", "aa",
	"aax", "abc", "acm", "adf", "adp", "ads", "adx", "aea", "afc", "aix",
	"al", "amf", "ams", "ans", "apl", "aptx", "aptxhd", "aqt", "art", "asc",
	"asf", "ast", "avc", "avi", "avifs", "avr", "avs", "avs2", "avs3", "bcstm",
	"bfstm", "binka", "bit", "bmv", "brstm", "c2", "cdata", "cdg", "cdxl", "cgi",
	"cif", "dat", "daud", "dav", "dbm", "dif", "digi", "divx", "diz", "dmf",
	"dsm", "dss", "dtk", "dtm", "dts", "dtshd", "dv", "eac3", "f4v", "fap",
	"far", "flm", "flv", "fsb", "fwse", "g722", "g723_1", "g729", "gdm", "genh",
	"gif", "gifv", "gsm", "h261", "h264", "h265", "h26l", "hca", "heics", "hevc",
	"ice", "idf", "ifv", "imf", "imx", "ipu", "ircam", "ism", "isma", "ismv",
	"it", "itgz", "itr", "itz", "ivr", "j2b", "j2k", "kux", "lvf", "m15",
	"m2ts", "m4v", "mac", "mca", "mcc", "mdgz", "mdl", "mdr", "mdz", "med",
	"mj2", "mjpeg", "mjpg", "mk3d", "mkv", "mlp", "mmcmp", "mms", "mo3", "mod",
	"mods", "moflex", "mov", "mp2", "mp4", "mpa", "mpc", "mpeg", "mpg", "mpl2",
	"mpo", "mptm", "msbc", "msf", "mt2", "mtaf", "mtm", "mts", "musx", "mvi",
	"mxg", "nist", "nst", "nut", "obu", "ogm", "ogv", "okt", "oma", "omg",
	"paf", "pjs", "plm", "ppm", "psm", "psp", "pt36", "ptm", "pvf", "qcif",
	"rco", "rcv", "rgb", "rt", "rsd", "rmvb", "rm", "rsd", "rso", "rt", "s3gz", "s3m",
	"s3r", "s3z", "sami", "sb", "sbc", "sbg", "scc", "sdr2", "sds", "sdx",
	"ser", "sf", "sfx", "sfx2", "sga", "shn", "sln", "son", "sph", "ss2",
	"st26", "stk", "stl", "stm", "stp", "str", "sup", "svag", "svs", "sw",
	"tak", "tco", "thd", "tp", "ts", "tta", "ty", "ty+", "ub", "ul",
	"ult", "umx", "uw", "v", "v210", "vag", "vc1", "rcv", "vob", "viv",
	"vpk", "vqe", "vqf", "vql", "vt", "webm", "wmv", "wow", "wsd", "xl",
	"xm", "xmgz", "xmr", "xmv", "xmz", "xpk", "xvag", "y4m", "yop", "yuv",
	"yuv10",
}

var ImageExtensions = []string{
	"360", "3fr", "aai", "ai", "ait", "arq", "arw", "avif", "avs", "bmp",
	"bmp2", "bmp3", "bpg", "btf", "ciff", "cr2", "cr3", "crm", "crw", "cs1",
	"dcp", "dcr", "dng", "dvb", "dvr-ms", "eip", "emf", "eps", "epsf", "erf",
	"exif", "exv", "f4a", "f4b", "f4p", "farbfeld", "fax", "fff", "fits", "fl32",
	"fla", "flif", "fpx", "gpr", "hdp", "heic", "heif", "hif", "icc", "icm",
	"iiq", "insp", "insv", "inx", "j2c", "jbig", "jng", "jp2", "jpc", "jpe",
	"jpeg", "jpf", "jpg", "jpm", "jpx", "jxl", "jxr", "k25", "kdc", "la",
	"lrv", "m4p", "max", "mef", "mie", "mif", "miff", "mng", "mos", "mqv",
	"mrw", "nef", "nksc", "nrw", "ofr", "orf", "ori", "pac", "pbm", "pct",
	"pef", "pfm", "pgm", "phm", "pict", "png", "pnm", "ppm", "ps", "psb",
	"psd", "psdt", "pspimage", "ptif", "qif", "qoi", "qt", "qti", "qtif", "raf",
	"raw", "rif", "riff", "rw2", "rwl", "rwz", "sr2", "srf", "srw", "swf",
	"tga", "thm", "tif", "tiff", "vrd", "wdp", "webp", "wmf", "x3f", "xcf",
	"xmp",
}

var TextExtensions = []string{
	"azw", "azw3", "azw4", "cbc", "chm", "djv", "djvu", "doc", "docx", "dot",
	"epub", "fb2", "fbz", "htmlz", "ind", "indd", "indt", "lit", "lrf", "md",
	"mobi", "odt", "pdb", "pdf", "pml", "pot", "pps", "ppt", "prc", "rb",
	"rtf", "snb", "tcr", "txtz", "vsd", "xls", "xlt",
}

var ArchiveExtensions = []string{
	"0", "0001", "001", "01", "1", "7z", "Z", "ace", "alz", "alzip",
	"arc", "arj", "b5i", "b6i", "bin", "br", "bz2", "cab", "cb7", "cba",
	"cbr", "cbt", "cbz", "ccd", "cdr", "cif", "cpio", "daa", "deb", "dmg",
	"exe", "gi", "gz", "img", "iso", "lha", "lzh", "lzma", "lzo", "lzx",
	"mdf", "msi", "nrg", "nsi", "nsis", "p01", "pak", "pdi", "r00", "r01",
	"rar", "rpm", "sit", "sitx", "tar", "tar.bz2", "tar.gz", "tar.xz", "tar.zst", "taz",
	"tbz2", "tgz", "toast", "txz", "tz", "tzst", "udf", "uif", "vcd", "wim",
	"xar", "xz", "z", "z00", "z01", "zip", "zipx", "zoo", "zst", "zstd",
}

var ComicExtensions = []string{
	"cbz", "cbr",
}

var (
	VideoExtensionMap   = make(map[string]bool)
	AudioExtensionMap   = make(map[string]bool)
	ImageExtensionMap   = make(map[string]bool)
	TextExtensionMap    = make(map[string]bool)
	ComicExtensionMap   = make(map[string]bool)
	ArchiveExtensionMap = make(map[string]bool)
	MediaExtensionMap   = make(map[string]bool)
)

func init() {
	for _, ext := range VideoExtensions {
		VideoExtensionMap["."+ext] = true
		MediaExtensionMap["."+ext] = true
	}
	for _, ext := range AudioExtensions {
		AudioExtensionMap["."+ext] = true
		MediaExtensionMap["."+ext] = true
	}
	for _, ext := range ImageExtensions {
		ImageExtensionMap["."+ext] = true
		MediaExtensionMap["."+ext] = true
	}
	for _, ext := range TextExtensions {
		TextExtensionMap["."+ext] = true
		MediaExtensionMap["."+ext] = true
	}
	for _, ext := range ComicExtensions {
		ComicExtensionMap["."+ext] = true
		MediaExtensionMap["."+ext] = true
	}
	for _, ext := range ArchiveExtensions {
		ArchiveExtensionMap["."+ext] = true
		MediaExtensionMap["."+ext] = true
	}
}

var SubtitleExtensions = []string{
	"srt", "vtt", "mks", "ass", "ssa", "lrc", "idx", "sub",
}

// UnreliableDurationFormats are formats known to have unreliable duration metadata
// (DVD, Blu-ray, camcorder formats, and older codecs)
// The int value is the estimated bitrate in bits per second for each format
var UnreliableDurationFormats = map[string]int{
	// DVD formats (lower bitrate, ~5-10 Mbps typical)
	".vob": 5000000, // DVD Video Object
	".ifo": 5000000, // DVD Information
	".vro": 5000000, // DVD Recording format

	// AVCHD / Camcorder formats (medium bitrate, ~10-20 Mbps typical)
	".m2t":  15000000, // MPEG-2 Transport Stream
	".m2ts": 15000000, // Blu-ray MPEG-2 Transport Stream
	".mts":  15000000, // AVCHD Video
	".mod":  10000000, // Canon/ JVC camcorder format
	".tod":  12000000, // JVC camcorder format

	// Older/lossy codecs (variable bitrate, ~2-8 Mbps typical)
	".divx": 4000000, // DivX codec
	".xvid": 4000000, // Xvid codec
	".rm":   2000000, // RealMedia
	".rmvb": 3000000, // RealMedia Variable Bitrate
	".wmv":  3000000, // Windows Media Video
	".asf":  3000000, // Advanced Systems Format

	// Blu-ray formats (high bitrate, ~20-40 Mbps typical)
	".avchd": 20000000, // AVCHD container
	".bdmv":  30000000, // Blu-ray Disc Movie
	".mpls":  30000000, // Blu-ray Playlist

	// Disc images (use average of contained formats)
	".iso": 8000000, // Disc image (average estimate)
}

// HasUnreliableDuration checks if a file extension is known to have unreliable duration metadata
func HasUnreliableDuration(ext string) bool {
	_, ok := UnreliableDurationFormats[strings.ToLower(ext)]
	return ok
}

// GetEstimatedBitrate returns the estimated bitrate for a format
// Returns 0 if the format is not in the unreliable formats map
func GetEstimatedBitrate(ext string) int {
	return UnreliableDurationFormats[strings.ToLower(ext)]
}

// Default bitrates for duration estimation (bits per second)
const (
	DefaultAudioBitrate = 256000  // 256 kbps
	DefaultVideoBitrate = 1500000 // 1500 kbps
)

// EstimateDurationFromSize estimates duration from file size and bitrate
// Returns duration in seconds
func EstimateDurationFromSize(size int64, isVideo bool) float64 {
	bitrate := DefaultAudioBitrate
	if isVideo {
		bitrate = DefaultVideoBitrate
	}
	return float64(size) / float64(bitrate) * 8
}

// EstimateDurationFromSizeWithFormat estimates duration from file size using format-specific bitrate
// Returns duration in seconds
func EstimateDurationFromSizeWithFormat(size int64, ext string) float64 {
	bitrate := GetEstimatedBitrate(ext)
	if bitrate <= 0 {
		// Fallback to default estimation
		return EstimateDurationFromSize(size, true)
	}
	return float64(size) / float64(bitrate) * 8
}

// GetDurationForTimeout returns a duration value suitable for timeout calculations.
// If the provided duration is valid (> 0), it returns it as-is.
// If duration is <= 0, it estimates from file size:
//   - For unreliable formats (DVD, Blu-ray, etc.), uses format-specific bitrate
//   - For other formats, uses default video bitrate
//
// Returns 0 if size is invalid (<= 0)
func GetDurationForTimeout(duration float64, size int64, ext string) float64 {
	if duration > 0 {
		return duration
	}
	if size <= 0 {
		return 0
	}
	return EstimateDurationFromSizeWithFormat(size, ext)
}

// ShouldOverrideDuration determines if reported duration should be overridden
// with an estimate based on file size. Returns true only when:
//   - File extension matches an unreliable format
//   - Reported duration is suspiciously low (< 2 minutes)
//   - Estimated duration is much higher (> 2 minutes)
func ShouldOverrideDuration(reportedDuration float64, size int64, ext string) (float64, bool) {
	if reportedDuration >= 120 {
		// Duration is >= 2 minutes, trust it
		return 0, false
	}
	if !HasUnreliableDuration(ext) {
		// Not an unreliable format, trust reported duration
		return 0, false
	}

	estimatedDuration := EstimateDurationFromSizeWithFormat(size, ext)
	if estimatedDuration <= 120 {
		// Estimated duration is also low, trust reported duration
		return 0, false
	}

	// Override with estimated duration
	return estimatedDuration, true
}

func GetTempDir() string {
	return os.TempDir()
}

func GetCattNowPlayingFile() string {
	return filepath.Join(os.TempDir(), "catt_playing")
}

func GetConfigDir() string {
	home, _ := os.UserHomeDir()
	if IsWindows {
		return filepath.Join(home, "AppData", "Roaming", "disco")
	}
	return filepath.Join(home, ".config", "disco")
}
