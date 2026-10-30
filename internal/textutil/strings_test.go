package textutil_test

import (
	"reflect"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/textutil"
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
		if got := textutil.CompareBlockStrings(tt.pattern, tt.value); got != tt.expected {
			t.Errorf("textutil.CompareBlockStrings(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.expected)
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
		if got := textutil.MatchesAny(tt.path, tt.patterns); got != tt.expected {
			t.Errorf("textutil.MatchesAny(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.expected)
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
		result := textutil.NaturalLess(tt.s1, tt.s2)
		if result != tt.expected {
			t.Errorf("textutil.NaturalLess(%q, %q) = %v, want %v", tt.s1, tt.s2, result, tt.expected)
		}
	}
}

func TestExtractNumbers(t *testing.T) {
	tests := []struct {
		input    string
		expected []textutil.Chunk
	}{
		{"abc123def", []textutil.Chunk{{"abc", 0, false}, {"", 123, true}, {"def", 0, false}}},
		{"123", []textutil.Chunk{{"", 123, true}}},
		{"abc", []textutil.Chunk{{"abc", 0, false}}},
	}
	for _, tt := range tests {
		got := textutil.ExtractNumbers(tt.input)
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("textutil.ExtractNumbers(%q) = %v, want %v", tt.input, got, tt.expected)
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
		if got := textutil.CleanString(tt.input); got != tt.expected {
			t.Errorf("textutil.CleanString(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCombine(t *testing.T) {
	if got := textutil.Combine("a", "b"); got != "a;b" {
		t.Errorf("textutil.Combine(a, b) = %q, want a;b", got)
	}
	if got := textutil.Combine("a", []string{"b", "c"}); got != "a;b;c" {
		t.Errorf("textutil.Combine(a, [b, c]) = %q, want a;b;c", got)
	}
	if got := textutil.Combine("a,b", "c;d"); got != "a;b;c;d" {
		t.Errorf("textutil.Combine(a,b, c;d) = %q, want a;b;c;d", got)
	}
}

func TestFromTimestampSeconds(t *testing.T) {
	if got := textutil.FromTimestampSeconds(":30"); got != 30 {
		t.Errorf("textutil.FromTimestampSeconds(:30) = %v, want 30", got)
	}
	if got := textutil.FromTimestampSeconds("1:30"); got != 90 {
		t.Errorf("textutil.FromTimestampSeconds(1:30) = %v, want 90", got)
	}
}

func TestExtractWords(t *testing.T) {
	input := "UniqueTerm, AnotherTerm! 123"
	expected := []string{"uniqueterm", "anotherterm", "123"}
	got := textutil.ExtractWords(input)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("textutil.ExtractWords() = %v, want %v", got, expected)
	}
}

func TestPathToSentence(t *testing.T) {
	input := "/home/user/movie_title.mp4"
	expected := "movie title mp4"
	if got := textutil.PathToSentence(input); got != expected {
		t.Errorf("textutil.PathToSentence(%q) = %q, want %q", input, got, expected)
	}
}

func TestIsDigit(t *testing.T) {
	if !textutil.IsDigit("123") {
		t.Error("textutil.IsDigit(123) should be true")
	}
	if textutil.IsDigit("12a") {
		t.Error("textutil.IsDigit(12a) should be false")
	}
	if textutil.IsDigit("") {
		t.Error("textutil.IsDigit('') should be false")
	}
}

func TestRemoveConsecutiveWhitespace(t *testing.T) {
	input := "hello   world \t  again"
	expected := "hello world again"
	if got := textutil.RemoveConsecutiveWhitespace(input); got != expected {
		t.Errorf("textutil.RemoveConsecutiveWhitespace(%q) = %q, want %q", input, got, expected)
	}
}

func TestRemoveConsecutives(t *testing.T) {
	if got := textutil.RemoveConsecutive("aaa", "a"); got != "a" {
		t.Errorf("textutil.RemoveConsecutive(aaa) = %q, want a", got)
	}
	if got := textutil.RemoveConsecutives("aaabbb", []string{"a", "b"}); got != "ab" {
		t.Errorf("textutil.RemoveConsecutives(aaabbb) = %q, want ab", got)
	}
}

func TestTitle(t *testing.T) {
	input := "hello world"
	expected := "Hello World"
	if got := textutil.Title(input); got != expected {
		t.Errorf("textutil.Title(%q) = %q, want %q", input, got, expected)
	}
	if got := textutil.Title(""); got != "" {
		t.Errorf("textutil.Title('') = %q, want ''", got)
	}
}

func TestSplitAndTrim(t *testing.T) {
	input := " a , b , c "
	expected := []string{"a", "b", "c"}
	got := textutil.SplitAndTrim(input, ",")
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("textutil.SplitAndTrim() = %v, want %v", got, expected)
	}
}

func TestEscapeXML(t *testing.T) {
	input := "< & > \" '"
	expected := "&lt; &amp; &gt; &quot; &apos;"
	if got := textutil.EscapeXML(input); got != expected {
		t.Errorf("textutil.EscapeXML() = %q, want %q", got, expected)
	}
}
