package goignore

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// this file was adapted from the go-gitignore package:
// https://github.com/sabhiram/go-gitignore/blob/525f6e181f062064d83887ed2530e3b1ba0bc95a/ignore_ported_test.go

/*
The MIT License (MIT)

Copyright (c) 2015 Shaba Abhiram

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

func ExampleCompileIgnoreLines() {
	ignoreObject := CompileIgnoreLines([]string{"node_modules", "*.out", "foo/*.c"})

	// You can test the ignoreObject against various paths using the
	// "Match()" interface method. This pretty much is up to
	// the users interpretation. In the case of a ".gitignore" file,
	// a "match" would indicate that a given path would be ignored.
	fmt.Println(ignoreObject.MatchesPath("node_modules/test/foo.js"))
	fmt.Println(ignoreObject.MatchesPath("node_modules2/test.out"))
	fmt.Println(ignoreObject.MatchesPath("test/foo.js"))

	// Output:
	// true <nil>
	// true <nil>
	// false <nil>
}

func ExampleCompileIgnoreFile() {
	ignoreObject, err := CompileIgnoreFile(".gitignore")

	// err returns an error from os.ReadFile()
	if err != nil {
		fmt.Println("Error reading .gitignore file:", err)
		return
	}

	// You can test the ignoreObject against various paths using the
	// "Match()" interface method.
	// int this example, we test paths against the .gitignore file of this package.
	fmt.Println(ignoreObject.MatchesPath("bin/goignore.so"))
	fmt.Println(ignoreObject.MatchesPath("goignore.test"))
	fmt.Println(ignoreObject.MatchesPath("go.mod"))

	// Output:
	// true <nil>
	// true <nil>
	// false <nil>
}

// Validate the correct handling of the negation operator "!"
func TestCompileIgnoreLines_HandleIncludePattern(t *testing.T) {
	ignoreObject := CompileIgnoreLines([]string{
		"/*",
		"!/foo",
		"/foo/*",
		"!/foo/bar",
	})

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("a"), "a should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("foo/baz"), "foo/baz should match")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("foo"), "foo should not match")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("/foo/bar"), "/foo/bar should not match")
}

// Validate the correct handling of leading / chars
func TestCompileIgnoreLines_HandleLeadingSlash(t *testing.T) {
	ignoreObject := CompileIgnoreLines([]string{
		"/a/b/c",
		"d/e/f",
		"/g",
	})

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("a/b/c"), "a/b/c should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("a/b/c/d"), "a/b/c/d should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("d/e/f"), "d/e/f should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("g"), "g should match")
}

// Validate the correct handling of files starting with # or !
func TestCompileIgnoreLines_HandleLeadingSpecialChars(t *testing.T) {
	ignoreObject := CompileIgnoreLines([]string{
		"# Comment",
		"\\#file.txt",
		"\\!file.txt",
		"file.txt",
	})

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("#file.txt"), "#file.txt should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("!file.txt"), "!file.txt should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("a/!file.txt"), "a/!file.txt should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("file.txt"), "file.txt should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("a/file.txt"), "a/file.txt should match")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("file2.txt"), "file2.txt should not match")

}

// Validate the correct handling matching files only within a given folder
func TestCompileIgnoreLines_HandleAllFilesInDir(t *testing.T) {
	ignoreObject := CompileIgnoreLines([]string{"Documentation/*.html"})

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("Documentation/git.html"), "Documentation/git.html should match")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("Documentation/ppc/ppc.html"), "Documentation/ppc/ppc.html should not match")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("tools/perf/Documentation/perf.html"), "tools/perf/Documentation/perf.html should not match")
}

// Validate the correct handling of "**"
func TestCompileIgnoreLines_HandleDoubleStar(t *testing.T) {
	ignoreObject := CompileIgnoreLines([]string{"**/foo", "bar", "baz/**"})

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("foo"), "foo should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("baz/foo"), "baz/foo should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("bar"), "bar should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("fizz/bar"), "fizz/bar should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("baz/buzz"), "baz/buzz should match")
}

// Validate the correct handling of leading slash
func TestCompileIgnoreLines_HandleLeadingSlashPath(t *testing.T) {
	object := CompileIgnoreLines([]string{"/*.c"})

	assert.NotNil(t, object, "Returned object should not be nil")

	assert.Equal(t, true, object.matchesPathNoError("hello.c"), "hello.c should match")
	assert.Equal(t, false, object.matchesPathNoError("foo/hello.c"), "foo/hello.c should not match")
}

func TestCompileIgnoreLines_CheckNestedDotFiles(t *testing.T) {
	lines := []string{
		"**/external/**/*.md",
		"**/external/**/*.json",
		"**/external/**/*.gzip",
		"**/external/**/.*ignore",

		"**/external/foobar/*.css",
		"**/external/barfoo/less",
		"**/external/barfoo/scss",
	}
	ignoreObject := CompileIgnoreLines(lines)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("external/foobar/angular.foo.css"), "external/foobar/angular.foo.css should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("external/barfoo/.gitignore"), "external/barfoo/.gitignore should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("external/barfoo/.bower.json"), "external/barfoo/.bower.json should match")
}

func TestCompileIgnoreLines_CarriageReturn(t *testing.T) {
	lines := []string{"abc/def\r", "a/b/c\r", "b\r"}
	ignoreObject := CompileIgnoreLines(lines)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("abc/def/child"), "abc/def/child should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("a/b/c/d"), "a/b/c/d should match")

	assert.Equal(t, false, ignoreObject.matchesPathNoError("abc"), "abc should not match")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("def"), "def should not match")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("bd"), "bd should not match")
}

func TestCompileIgnoreLines_WindowsPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		return
	}
	lines := []string{"abc/def", "a/b/c", "b"}
	ignoreObject := CompileIgnoreLines(lines)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("abc\\def\\child"), "abc\\def\\child should match")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("a\\b\\c\\d"), "a\\b\\c\\d should match")
}

func TestWildCardFiles(t *testing.T) {
	gitIgnore := []string{"*.swp", "/foo/*.wat", "bar/*.txt"}
	ignoreObject := CompileIgnoreLines(gitIgnore)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")

	// Paths which are targeted by the above "lines"
	assert.Equal(t, true, ignoreObject.matchesPathNoError("yo.swp"), "should ignore all swp files")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("something/else/but/it/hasyo.swp"), "should ignore all swp files in other directories")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("foo/bar.wat"), "should ignore all wat files in foo - nonpreceding /")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("/foo/something.wat"), "should not ignore all wat files in foo - preceding /")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("bar/something.txt"), "should ignore all txt files in bar - nonpreceding /")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("/bar/somethingelse.txt"), "should not ignore all txt files in bar - preceding /")

	// Paths which are not targeted by the above "lines"
	assert.Equal(t, false, ignoreObject.matchesPathNoError("something/not/infoo/wat.wat"), "wat files should only be ignored in foo")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("something/not/infoo/wat.txt"), "txt files should only be ignored in bar")
}

func TestPrecedingSlash(t *testing.T) {
	gitIgnore := []string{"/foo", "bar/"}
	ignoreObject := CompileIgnoreLines(gitIgnore)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("foo/bar.wat"), "should ignore all files in foo - nonpreceding /")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("/foo/something.txt"), "should not ignore all files in foo - preceding /")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("bar/something.txt"), "should ignore all files in bar - nonpreceding /")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("/bar/somethingelse.go"), "should not ignore all files in bar - preceding /")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("/boo/something/bar/boo.txt"), "should not ignore all files if bar is a sub directory")

	assert.Equal(t, false, ignoreObject.matchesPathNoError("something/foo/something.txt"), "should only ignore top level foo directories - not nested")
}

func TestDirOnlyMatching(t *testing.T) {
	gitIgnore := []string{"foo/", "bar/"}
	ignoreObject := CompileIgnoreLines(gitIgnore)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("foo/"), "should match foo directory")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("bar/"), "should match bar directory")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("foo"), "should not match foo file")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("bar"), "should not match bar file")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("foo/bar"), "should match nested files in foo")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("bar/foo"), "should match nested files in bar")
}

func TestCharacterClasses(t *testing.T) {
	gitIgnore := []string{"[a-zA-Z*!]-files"}
	ignoreObject := CompileIgnoreLines(gitIgnore)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("a-files"), "should match a-files")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("g-files"), "should match g-files")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("z-files"), "should match z-files")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("!-files"), "should match !-files")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("*-files"), "should match *-files")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("8-files"), "should not match 8-files")

	gitIgnore = []string{"[!a-zA-Z*!]-files"}
	ignoreObject = CompileIgnoreLines(gitIgnore)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")

	assert.Equal(t, false, ignoreObject.matchesPathNoError("a-files"), "should not match a-files")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("g-files"), "should not match g-files")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("z-files"), "should not match z-files")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("!-files"), "should not match !-files")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("*-files"), "should not match *-files")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("8-files"), "should match 8-files")

	gitIgnore = []string{"[]-]"}
	ignoreObject = CompileIgnoreLines(gitIgnore)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("]"), "should match ]")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("-"), "should match -")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("[]-]"), "should not match []-]")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("[]-]"), "should not match -]")

	gitIgnore = []string{"[[:digit:]].txt", "[:alpha:].txt"}
	ignoreObject = CompileIgnoreLines(gitIgnore)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")

	assert.Equal(t, true, ignoreObject.matchesPathNoError("6.txt"), "should match 6.txt")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("z.txt"), "should not match z.txt")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("a.txt"), "should match a.txt")
}

func TestUnclosedCharacterClass(t *testing.T) {
	gitIgnore := []string{"*[*"}
	ignoreObject := CompileIgnoreLines(gitIgnore)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("["), "should not match [")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("*["), "should not match *[")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("[*"), "should not match [*")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("*[*"), "should not match *[*")

	gitIgnore = []string{"[a-z][[]A-Z*-files"}
	ignoreObject = CompileIgnoreLines(gitIgnore)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("a[A-Z-files"), "should match a[A-Z-files")
}

func TestStarExponentialBehaviour(t *testing.T) {
	gitIgnore := []string{"*a*a*a*a*a*a*a*a*a*a*a*a"}
	ignoreObject := CompileIgnoreLines(gitIgnore)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"), "should not match")
}

func TestStarStarExponentialBehaviour(t *testing.T) {
	gitIgnore := []string{"**/a/**/a/**/a/**/a/**/a/**/a/**/a/**/a/**/a/**/a"}
	ignoreObject := CompileIgnoreLines(gitIgnore)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/b"), "should match")
}

func TestStarFilepath(t *testing.T) {
	gitIgnore := []string{"\\*"}
	ignoreObject := CompileIgnoreLines(gitIgnore)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("*"), "should match")
}

func TestEscaping(t *testing.T) {
	gitIgnore := []string{"\\[hello", "bye[\\]"}
	ignoreObject := CompileIgnoreLines(gitIgnore)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("[hello"), "should match [hello")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("bye[]"), "should not match bye[]")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("bye["), "should not match bye[")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("bye[\\]"), "should not match bye[\\]")
}

func TestFolders(t *testing.T) {
	gitIgnore := []string{"Folder/"}
	ignoreObject := CompileIgnoreLines(gitIgnore)

	assert.NotNil(t, ignoreObject, "Returned object should not be nil")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("Folder/Folder"), "should match Folder/Folder")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("Folder/Buzz"), "should match Folder/Buzz")
	assert.Equal(t, true, ignoreObject.matchesPathNoError("Bar/Folder/"), "should match Bar/Folder/")
	assert.Equal(t, false, ignoreObject.matchesPathNoError("Fizz/Folder"), "should not match Fizz/Folder")
}

// Test for both mySplit() and mySplitBuf()
func TestMySplit(t *testing.T) {
	type TestCase struct {
		str       string
		separator byte
		expected  []string
	}

	repeatSlice := func(s string, numTimes int) []string {
		out := make([]string, numTimes)
		for i := range out {
			out[i] = s
		}
		return out
	}

	tests := []TestCase{
		{"this is a test", ' ', []string{"this", "is", "a", "test"}},
		{"dontsplit", ' ', []string{"dontsplit"}},
		{"", ' ', []string{}},
		{"aaaaa", 'a', []string{}},
		// Make sure we don't crash when splitting the max amount of path components in mySplitBuf()
		{strings.Repeat("a/", bufferLengthForPathComponents()), '/', repeatSlice("a", bufferLengthForPathComponents())},
	}

	// mySplitBuf expects a buffer slice of sufficient length.
	buffer := make([]string, bufferLengthForPathComponents())

	for _, test := range tests {
		result := mySplitBuf(test.str, test.separator, buffer)
		assert.Equal(t, test.expected, result)

		result = mySplit(test.str, test.separator)
		assert.Equal(t, test.expected, result)
	}
}

func TestMaxPathLengthError(t *testing.T) {
	g := CompileIgnoreLines([]string{""})

	// The path length limit
	longPath := strings.Repeat("a", maxPathLength())
	_, err := g.MatchesPath(longPath)
	assert.Equal(t, nil, err)

	// 1 character above the path length limit
	_, err = g.MatchesPath(longPath + "a")
	assert.NotEqual(t, nil, err)
}

func FuzzStringMatch(f *testing.F) {
	f.Add("hello, world!", "hell*[oasd], [[:alpha:]]orld!")
	f.Add("hello, world!", "hell*[!asd], [![:digit:]]orld!")
	f.Fuzz(func(t *testing.T, str string, pattern string) {
		stringMatch(str, pattern)
	})
}

func FuzzWhole(f *testing.F) {
	f.Add("hell*[oasd], [[:alpha:]]orld!", "hello, world!")
	f.Add("hell*[!asd], [![:digit:]]orld!", "hello, world!")
	f.Fuzz(func(t *testing.T, ignore string, path string) {
		ignoreObject := CompileIgnoreLines(strings.Split(ignore, "/n"))

		if ignoreObject == nil {
			t.Fail()
		}
		ignoreObject.MatchesPath(path)
	})
}
