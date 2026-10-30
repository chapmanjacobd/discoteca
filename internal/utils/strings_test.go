package utils_test

import (
	"reflect"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
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
		if got := utils.CompareBlockStrings(tt.pattern, tt.value); got != tt.expected {
			t.Errorf("utils.CompareBlockStrings(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.expected)
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
		if got := utils.MatchesAny(tt.path, tt.patterns); got != tt.expected {
			t.Errorf("utils.MatchesAny(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.expected)
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
		result := utils.NaturalLess(tt.s1, tt.s2)
		if result != tt.expected {
			t.Errorf("utils.NaturalLess(%q, %q) = %v, want %v", tt.s1, tt.s2, result, tt.expected)
		}
	}
}

func TestExtractNumbers(t *testing.T) {
	tests := []struct {
		input    string
		expected []utils.Chunk
	}{
		{"abc123def", []utils.Chunk{{"abc", 0, false}, {"", 123, true}, {"def", 0, false}}},
		{"123", []utils.Chunk{{"", 123, true}}},
		{"abc", []utils.Chunk{{"abc", 0, false}}},
	}
	for _, tt := range tests {
		got := utils.ExtractNumbers(tt.input)
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("utils.ExtractNumbers(%q) = %v, want %v", tt.input, got, tt.expected)
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
		if got := utils.CleanString(tt.input); got != tt.expected {
			t.Errorf("utils.CleanString(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCombine(t *testing.T) {
	if got := utils.Combine("a", "b"); got != "a;b" {
		t.Errorf("utils.Combine(a, b) = %q, want a;b", got)
	}
	if got := utils.Combine("a", []string{"b", "c"}); got != "a;b;c" {
		t.Errorf("utils.Combine(a, [b, c]) = %q, want a;b;c", got)
	}
	if got := utils.Combine("a,b", "c;d"); got != "a;b;c;d" {
		t.Errorf("utils.Combine(a,b, c;d) = %q, want a;b;c;d", got)
	}
}

func TestFromTimestampSeconds(t *testing.T) {
	if got := utils.FromTimestampSeconds(":30"); got != 30 {
		t.Errorf("utils.FromTimestampSeconds(:30) = %v, want 30", got)
	}
	if got := utils.FromTimestampSeconds("1:30"); got != 90 {
		t.Errorf("utils.FromTimestampSeconds(1:30) = %v, want 90", got)
	}
}

func TestExtractWords(t *testing.T) {
	input := "UniqueTerm, AnotherTerm! 123"
	expected := []string{"uniqueterm", "anotherterm", "123"}
	got := utils.ExtractWords(input)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("utils.ExtractWords() = %v, want %v", got, expected)
	}
}

func TestPathToSentence(t *testing.T) {
	input := "/home/user/movie_title.mp4"
	expected := "movie title mp4"
	if got := utils.PathToSentence(input); got != expected {
		t.Errorf("utils.PathToSentence(%q) = %q, want %q", input, got, expected)
	}
}

func TestIsDigit(t *testing.T) {
	if !utils.IsDigit("123") {
		t.Error("utils.IsDigit(123) should be true")
	}
	if utils.IsDigit("12a") {
		t.Error("utils.IsDigit(12a) should be false")
	}
	if utils.IsDigit("") {
		t.Error("utils.IsDigit('') should be false")
	}
}

func TestRemoveConsecutiveWhitespace(t *testing.T) {
	input := "hello   world \t  again"
	expected := "hello world again"
	if got := utils.RemoveConsecutiveWhitespace(input); got != expected {
		t.Errorf("utils.RemoveConsecutiveWhitespace(%q) = %q, want %q", input, got, expected)
	}
}

func TestRemoveConsecutives(t *testing.T) {
	if got := utils.RemoveConsecutive("aaa", "a"); got != "a" {
		t.Errorf("utils.RemoveConsecutive(aaa) = %q, want a", got)
	}
	if got := utils.RemoveConsecutives("aaabbb", []string{"a", "b"}); got != "ab" {
		t.Errorf("utils.RemoveConsecutives(aaabbb) = %q, want ab", got)
	}
}

func TestTitle(t *testing.T) {
	input := "hello world"
	expected := "Hello World"
	if got := utils.Title(input); got != expected {
		t.Errorf("utils.Title(%q) = %q, want %q", input, got, expected)
	}
	if got := utils.Title(""); got != "" {
		t.Errorf("utils.Title('') = %q, want ''", got)
	}
}

func TestSplitAndTrim(t *testing.T) {
	input := " a , b , c "
	expected := []string{"a", "b", "c"}
	got := utils.SplitAndTrim(input, ",")
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("utils.SplitAndTrim() = %v, want %v", got, expected)
	}
}

func TestEscapeXML(t *testing.T) {
	input := "< & > \" '"
	expected := "&lt; &amp; &gt; &quot; &apos;"
	if got := utils.EscapeXML(input); got != expected {
		t.Errorf("utils.EscapeXML() = %q, want %q", got, expected)
	}
}
