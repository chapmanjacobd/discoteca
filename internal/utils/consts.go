package utils

import (
	"os"
	"path/filepath"
	"runtime"
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
	"nksc", "vrd", "xmp", "la", "ofr", "pac", "riff", "rif", "wav", "webp", "wv", "djvu", "djv", "dvr-ms",
	"insv", "inx", "swf", "exif", "eip", "pspimage", "fax", "farbfeld", "fits", "fl32", "jbig",
	"pbm", "pfm", "pgm", "phm", "pnm", "ppm", "ptif", "qoi", "tga",
}

var TextExtensions = []string{
	"epub", "mobi", "pdf", "azw", "azw3", "fb2", "djvu", "cbz", "cbr", "zim",
}

var (
	VideoExtensionMap   = make(map[string]bool)
	AudioExtensionMap   = make(map[string]bool)
	ImageExtensionMap   = make(map[string]bool)
	TextExtensionMap    = make(map[string]bool)
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
	for _, ext := range ArchiveExtensions {
		ArchiveExtensionMap["."+ext] = true
		MediaExtensionMap["."+ext] = true
	}
}

var SubtitleExtensions = []string{
	"srt", "vtt", "mks", "ass", "ssa", "lrc", "idx", "sub",
}

var ArchiveExtensions = []string{
	"7z", "arj", "arc", "adf", "br", "bz2", "gz", "iso", "lha", "lzh", "lzx", "pak", "rar", "sit", "tar", "tar.bz2", "tar.gz", "tar.xz", "tar.zst", "tbz2", "tgz", "txz", "tzst", "xz", "zoo", "zip", "zst", "zstd",
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
