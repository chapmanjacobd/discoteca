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
	"mka", "opus", "oga", "ogg", "mp3", "mpga", "m2a", "m4a", "m4r", "caf", "m4b", "flac", "wav", "pcm", "aif", "aiff", "wma", "aac", "aa3", "ac3", "ape", "dsf", "dff",
}

var VideoExtensions = []string{
	"str", "aa", "aax", "acm", "adf", "adp", "asf", "dtk", "ads", "ss2", "adx", "aea", "afc", "aix", "al", "apl", "avifs", "gif", "gifv",
	"mac", "aptx", "aptxhd", "aqt", "ast", "obu", "avi", "avr", "avs", "avs2", "avs3", "bfstm", "bcstm", "binka",
	"bit", "bmv", "brstm", "cdg", "cdxl", "xl", "c2", "302", "daud", "str", "adp", "dav", "dss", "dts", "dtshd", "dv",
	"dif", "divx", "cdata", "eac3", "paf", "fap", "flm", "flv", "fsb", "fwse", "g722", "722", "tco", "rco", "heics",
	"g723_1", "g729", "genh", "gsm", "h261", "h26l", "h264", "264", "avc", "mts", "m2ts", "hca", "hevc", "h265", "265", "idf",
	"ifv", "cgi", "ipu", "sf", "ircam", "ivr", "kux", "669", "abc", "amf", "ams", "dbm", "dmf", "dsm", "far", "it", "mdl",
	"med", "mod", "mt2", "mtm", "okt", "psm", "ptm", "s3m", "stm", "ult", "umx", "xm", "itgz", "itr", "itz",
	"mdgz", "mdr", "mdz", "s3gz", "s3r", "s3z", "xmgz", "xmr", "xmz", "669", "amf", "ams", "dbm", "digi", "dmf",
	"dsm", "dtm", "far", "gdm", "ice", "imf", "it", "j2b", "m15", "mdl", "med", "mmcmp", "mms", "mo3", "mod", "mptm",
	"mt2", "mtm", "nst", "okt", "ogm", "ogv", "plm", "ppm", "psm", "pt36", "ptm", "s3m", "sfx", "sfx2", "st26", "stk", "stm",
	"stp", "ult", "umx", "wow", "xm", "xpk", "flv", "dat", "lvf", "m4v", "mkv", "ts", "tp", "mk3d", "webm", "mca", "mcc",
	"mjpg", "mjpeg", "mpg", "mpeg", "mpo", "j2k", "mlp", "mods", "moflex", "mov", "mp4", "3g2", "3gp2", "3gp", "3gpp", "3g2", "mj2", "psp",
	"ism", "ismv", "isma", "f4v", "mp2", "mpa", "mpc", "mjpg", "mpl2", "msf", "mtaf", "ul", "musx", "mvi", "mxg",
	"v", "nist", "sph", "nut", "obu", "oma", "omg", "pjs", "pvf", "yuv", "cif", "qcif", "rgb", "rt", "rsd", "rmvb", "rm",
	"rsd", "rso", "sw", "sb", "sami", "sbc", "msbc", "sbg", "scc", "sdr2", "sds", "sdx", "ser", "sga", "shn", "vb", "son", "imx",
	"sln", "mjpg", "stl", "sup", "svag", "svs", "tak", "thd", "tta", "ans", "art", "asc", "diz", "ice", "vt", "ty", "ty+", "uw", "ub",
	"v210", "yuv10", "vag", "vc1", "rcv", "vob", "viv", "vpk", "vqf", "vql", "vqe", "wmv", "wsd", "xmv", "xvag", "yop", "y4m",
}

var ImageExtensions = []string{
	"aai", "ai", "ait", "avs", "bpg", "png", "arq", "arw", "cr2", "cs1", "dcp", "dng", "eps", "epsf", "ps", "erf", "exv", "fff",
	"gpr", "hdp", "wdp", "jxr", "iiq", "insp", "jpeg", "jpg", "jpe", "mef", "mie", "mos", "mrw", "nef", "nrw", "orf",
	"ori", "pef", "psd", "psb", "psdt", "raf", "raw", "rw2", "rwl", "sr2", "srw", "thm", "tiff", "tif", "x3f", "flif",
	"icc", "icm", "avif", "heic", "heif", "hif", "jp2", "jpf", "jpm", "jpx", "j2c", "jpc", "3fr", "btf", "dcr", "k25",
	"kdc", "miff", "mif", "rwz", "srf", "xcf", "bpg", "doc", "dot", "fla", "fpx", "max", "ppt", "pps", "pot", "vsd", "xls",
	"xlt", "pict", "pct", "360", "dvb", "f4a", "f4b", "f4p", "lrv", "bmp", "bmp2", "bmp3", "jng", "mng", "emf", "wmf",
	"m4p", "qt", "mqv", "qtif", "qti", "qif", "cr3", "crm", "jxl", "crw", "ciff", "ind", "indd", "indt",
	"nksc", "vrd", "xmp", "la", "ofr", "pac", "riff", "rif", "webp", "wv", "djvu", "djv", "dvr-ms",
	"insv", "inx", "swf", "exif", "eip", "pspimage", "fax", "farbfeld", "fits", "fl32", "jbig",
	"pbm", "pfm", "pgm", "phm", "pnm", "ppm", "ptif", "qoi", "tga",
}

var TextExtensions = []string{
	"txt", "epub", "mobi", "pdf", "azw", "azw3", "fb2", "djvu", "zim",
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

var ArchiveExtensions = []string{
	"7z", "arj", "arc", "adf", "br", "bz2", "gz", "iso", "lha", "lzh", "lzx", "pak", "rar", "sit", "tar", "tar.bz2", "tar.gz", "tar.xz", "tar.zst", "tbz2", "tgz", "txz", "tzst", "xz", "zoo", "zip", "zst", "zstd", "cbz", "cbr",
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
