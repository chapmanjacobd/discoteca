package utils

import (
	"reflect"
	"testing"
)

func TestCompareBlockStrings(t *testing.T) {
	tests := []struct {
		pattern  string
		value    string
		expected bool
	}{
		{"abc%", "abcdef", true},
		{"%def", "abcdef", true},
		{"%bc%", "abcdef", true},
		{"a%f", "abcdef", true},
		{"missing", "abcdef", false},
	}
	for _, tt := range tests {
		if got := CompareBlockStrings(tt.pattern, tt.value); got != tt.expected {
			t.Errorf("CompareBlockStrings(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.expected)
		}
	}
}

func TestMatchesAny(t *testing.T) {
	tests := []struct {
		path     string
		patterns []string
		expected bool
	}{
		{"/home/user/movie.mp4", []string{"%.mp4"}, true},
		{"/home/user/movie.mp4", []string{"%.mkv"}, false},
		{"/home/user/movie.mp4", []string{"%user%"}, true},
		{"/home/user/movie.mp4", []string{"%missing%"}, false},
	}
	for _, tt := range tests {
		if got := MatchesAny(tt.path, tt.patterns); got != tt.expected {
			t.Errorf("MatchesAny(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.expected)
		}
	}
}

func TestNaturalLess(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected bool
	}{
		{"file1.txt", "file2.txt", true},
		{"file2.txt", "file1.txt", false},
		{"file1.txt", "file10.txt", true},
		{"file10.txt", "file2.txt", false},
		{"Season 1 Episode 1", "Season 1 Episode 10", true},
		{"S01E01", "S01E02", true},
		{"S01E02", "S01E01", false},
		{"S01E09", "S01E10", true},
	}

	for _, tt := range tests {
		result := NaturalLess(tt.s1, tt.s2)
		if result != tt.expected {
			t.Errorf("NaturalLess(%q, %q) = %v, want %v", tt.s1, tt.s2, result, tt.expected)
		}
	}
}

func TestExtractNumbers(t *testing.T) {
	tests := []struct {
		input    string
		expected []chunk
	}{
		{"abc123def", []chunk{{"abc", 0, false}, {"", 123, true}, {"def", 0, false}}},
		{"123", []chunk{{"", 123, true}}},
		{"abc", []chunk{{"abc", 0, false}}},
	}
	for _, tt := range tests {
		got := extractNumbers(tt.input)
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("extractNumbers(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestCleanString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello_world", "hello_world"},
		{"hello.world", "hello.world"},
		{"hello (world)", "hello"},
		{"Hello & World!", "Hello World"},
	}
	for _, tt := range tests {
		if got := CleanString(tt.input); got != tt.expected {
			t.Errorf("CleanString(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestShorten(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"hello world", 5, "hell…"},
		{"hello world", 20, "hello world"},
		{"こんにちは", 6, "こん…"},
	}
	for _, tt := range tests {
		if got := Shorten(tt.input, tt.width); got != tt.expected {
			t.Errorf("Shorten(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.expected)
		}
	}
}

func TestShortenMiddle(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"hello world", 8, "hel...ld"},
		{"こんにちは世界", 8, "こ...界"},
	}
	for _, tt := range tests {
		if got := ShortenMiddle(tt.input, tt.width); got != tt.expected {
			t.Errorf("ShortenMiddle(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.expected)
		}
	}
}

func TestStripEnclosingQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{"'hello'", "hello"},
		{"«hello»", "hello"},
		{`"'hello'"`, "hello"},
	}
	for _, tt := range tests {
		if got := StripEnclosingQuotes(tt.input); got != tt.expected {
			t.Errorf("StripEnclosingQuotes(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCombine(t *testing.T) {
	if got := Combine("a", "b"); got != "a;b" {
		t.Errorf("Combine(a, b) = %q, want a;b", got)
	}
	if got := Combine("a", []string{"b", "c"}); got != "a;b;c" {
		t.Errorf("Combine(a, [b, c]) = %q, want a;b;c", got)
	}
	if got := Combine("a,b", "c;d"); got != "a;b;c;d" {
		t.Errorf("Combine(a,b, c;d) = %q, want a;b;c;d", got)
	}
}

func TestFromTimestampSeconds(t *testing.T) {
	if got := FromTimestampSeconds(":30"); got != 30 {
		t.Errorf("FromTimestampSeconds(:30) = %v, want 30", got)
	}
	if got := FromTimestampSeconds("1:30"); got != 90 {
		t.Errorf("FromTimestampSeconds(1:30) = %v, want 90", got)
	}
}

func TestPartialStartswith(t *testing.T) {
	list := []string{"daily", "weekly", "monthly"}
	if got, _ := PartialStartswith("da", list); got != "daily" {
		t.Errorf("PartialStartswith(da) = %q, want daily", got)
	}
}

func TestGlobMatchAny(t *testing.T) {
	if !GlobMatchAny("test", []string{"*test*"}) {
		t.Error("GlobMatchAny failed")
	}
}

func TestGlobMatchAll(t *testing.T) {
	if !GlobMatchAll("test", []string{"*test*", "t*"}) {
		t.Error("GlobMatchAll failed")
	}
}

func TestDurationShort(t *testing.T) {
	if got := DurationShort(60); got != "1 minute" {
		t.Errorf("DurationShort(60) = %q, want 1 minute", got)
	}
}

func TestExtractWords(t *testing.T) {
	input := "UniqueTerm, AnotherTerm! 123"
	expected := []string{"uniqueterm", "anotherterm", "123"}
	got := ExtractWords(input)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("ExtractWords() = %v, want %v", got, expected)
	}
}

func TestSafeJSONLoads(t *testing.T) {
	input := `{"a": 1}`
	got := SafeJSONLoads(input)
	expected := map[string]any{"a": float64(1)}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("SafeJSONLoads() = %v, want %v", got, expected)
	}
}

func TestLoadString(t *testing.T) {
	input := `{'a': 1}`
	got := LoadString(input)
	expected := map[string]any{"a": float64(1)}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("LoadString() = %v, want %v", got, expected)
	}

	if got := LoadString("just string"); got != "just string" {
		t.Errorf("LoadString() = %v, want just string", got)
	}
}

func TestPathToSentence(t *testing.T) {
	input := "/home/user/movie_title.mp4"
	expected := "movie title mp4"
	if got := PathToSentence(input); got != expected {
		t.Errorf("PathToSentence(%q) = %q, want %q", input, got, expected)
	}
}

func TestIsGenericTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", true},
		{"Chapter 1", true},
		{"Scene 1", true},
		{"Untitled Chapter", true},
		{"123", true},
		{"00:01:02", true},
		{"Real Title", false},
	}
	for _, tt := range tests {
		if got := IsGenericTitle(tt.input); got != tt.expected {
			t.Errorf("IsGenericTitle(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestIsDigit(t *testing.T) {
	if !IsDigit("123") {
		t.Error("IsDigit(123) should be true")
	}
	if IsDigit("12a") {
		t.Error("IsDigit(12a) should be false")
	}
	if IsDigit("") {
		t.Error("IsDigit('') should be false")
	}
}

func TestIsTimecodeLike(t *testing.T) {
	if !IsTimecodeLike("00:01:02") {
		t.Error("IsTimecodeLike(00:01:02) should be true")
	}
	if !IsTimecodeLike("123") {
		t.Error("IsTimecodeLike(123) should be true")
	}
	if IsTimecodeLike("abc") {
		t.Error("IsTimecodeLike(abc) should be false")
	}
}

func TestRemoveConsecutiveWhitespace(t *testing.T) {
	input := "hello   world \t  again"
	expected := "hello world again"
	if got := RemoveConsecutiveWhitespace(input); got != expected {
		t.Errorf("RemoveConsecutiveWhitespace(%q) = %q, want %q", input, got, expected)
	}
}

func TestRemoveConsecutives(t *testing.T) {
	if got := RemoveConsecutive("aaa", "a"); got != "a" {
		t.Errorf("RemoveConsecutive(aaa) = %q, want a", got)
	}
	if got := RemoveConsecutives("aaabbb", []string{"a", "b"}); got != "ab" {
		t.Errorf("RemoveConsecutives(aaabbb) = %q, want ab", got)
	}
}

func TestRemovePrefixesSuffixes(t *testing.T) {
	if got := RemovePrefixes("prepretest", []string{"pre"}); got != "test" {
		t.Errorf("RemovePrefixes(prepretest) = %q, want test", got)
	}
	if got := RemoveSuffixes("testsuf", []string{"suf"}); got != "test" {
		t.Errorf("RemoveSuffixes(testsuf) = %q, want test", got)
	}
}

func TestUnParagraph(t *testing.T) {
	input := "“Hello” world …"
	expected := "'Hello' world ..."
	if got := UnParagraph(input); got != expected {
		t.Errorf("UnParagraph(%q) = %q, want %q", input, got, expected)
	}
}

func TestTitle(t *testing.T) {
	input := "hello world"
	expected := "Hello World"
	if got := Title(input); got != expected {
		t.Errorf("Title(%q) = %q, want %q", input, got, expected)
	}
	if got := Title(""); got != "" {
		t.Errorf("Title('') = %q, want ''", got)
	}
}

func TestSplitAndTrim(t *testing.T) {
	input := " a , b , c "
	expected := []string{"a", "b", "c"}
	got := SplitAndTrim(input, ",")
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("SplitAndTrim() = %v, want %v", got, expected)
	}
}

func TestRemoveExcessiveLinebreaks(t *testing.T) {
	input := "a\n\n\n\nb"
	expected := "a\n\nb"
	if got := RemoveExcessiveLinebreaks(input); got != expected {
		t.Errorf("RemoveExcessiveLinebreaks() = %q, want %q", got, expected)
	}
}

func TestLastChars(t *testing.T) {
	if got := LastChars("a.b.c"); got != "c" {
		t.Errorf("LastChars(a.b.c) = %q, want c", got)
	}
}

func TestFtsQuote(t *testing.T) {
	input := []string{"hello", "world AND universe"}
	expected := []string{"\"hello\"", "world AND universe"}
	got := FtsQuote(input)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("FtsQuote() = %v, want %v", got, expected)
	}
}

func TestEscapeXML(t *testing.T) {
	input := "< & > \" '"
	expected := "&lt; &amp; &gt; &quot; &apos;"
	if got := EscapeXML(input); got != expected {
		t.Errorf("EscapeXML() = %q, want %q", got, expected)
	}
}
