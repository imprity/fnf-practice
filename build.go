//go:build ignore

package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	FlagDebug = flag.Bool("debug", false, "disable inlining and optimization")
	FlagPprof = flag.Bool("pprof", false, "enable pprof")
	FlagDemo  = flag.Bool("demo", false, "enable demo recording for main app")
)

func init() {
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		_, scriptName := filepath.Split(os.Args[0])

		if _, scriptFile, _, ok := runtime.Caller(0); ok {
			_, scriptName = filepath.Split(scriptFile)
		}

		fmt.Fprintf(out, "Usage of %s:\n", scriptName)
		fmt.Fprintf(out, "\n")
		fmt.Fprintf(out, "go run %s [flag...] [target]\n", scriptName)
		fmt.Fprintf(out, "\n")
		fmt.Fprintf(out, "valid targets:\n")
		fmt.Fprintf(out, "  font-gen\n")
		fmt.Fprintf(out, "  release\n")
		fmt.Fprintf(out, "  all\n")
		fmt.Fprintf(out, "\n")

		fmt.Fprintf(out, "flags:\n")
		flag.PrintDefaults()
	}
}

type SimpleError struct {
	IsFail   bool
	ExitCode int
}

var ErrorDefault = SimpleError{
	IsFail:   true,
	ExitCode: 69,
}

var ErrorOk = SimpleError{
	IsFail: false,
}

func SimpleErrorFromError(err error) SimpleError {
	if err == nil {
		return SimpleError{}
	}
	var exitErr *exec.ExitError
	if ok := errors.As(err, &exitErr); ok {
		return SimpleError{
			IsFail:   true,
			ExitCode: exitErr.ExitCode(),
		}
	} else {
		return ErrorDefault
	}
}

var GIT_TAG_VERSION string

var (
	ErrLogger  = log.New(os.Stderr, "[ FAIL! ] : ", log.Lshortfile)
	WarnLogger = log.New(os.Stderr, "[ WARN! ] : ", log.Lshortfile)
	Logger     = log.New(os.Stdout, "", 0)
)

func main() {
	flag.Parse()
	target := flag.Arg(0)

	// get git tag string
	{
		Logger.Print("getting git version string")

		if gitTag, err := GetGitTagVersion(); err.IsFail {
			GIT_TAG_VERSION = "unknown"
		} else {
			GIT_TAG_VERSION = gitTag
		}

		Logger.Print("writing it to git_tag.txt")
		if err := os.WriteFile("git_tag.txt", []byte(GIT_TAG_VERSION), 0664); err != nil {
			ErrLogger.Print("failed to write to git_tag.txt : %v", err)
			CrashOnError(ErrorDefault)
		}
	}

	buildMain := func() SimpleError {
		src := "main.go"
		dst := "fnf-practice"
		dst = AddExeIfWindows(dst)

		return BuildApp(
			src, dst, *FlagDebug, *FlagPprof, *FlagDemo,
		)
	}

	buildFontGen := func() SimpleError {
		src := "font_gen.go"
		dst := "font-gen"
		dst = AddExeIfWindows(dst)

		if *FlagDemo {
			WarnLogger.Print("demo flag is ignored for font-gen")
		}

		return BuildApp(
			src, dst, *FlagDebug, *FlagPprof, false,
		)
	}

	buildRelease := func() SimpleError {
		if runtime.GOOS != "windows" {
			ErrLogger.Print("building release is only supported on windows")
			return ErrorDefault
		}

		// delete release folder
		Logger.Print("deleting release folder")
		if err := os.RemoveAll("release"); err != nil {
			ErrLogger.Print("failed to delete release folder : %v", err)
			return ErrorDefault
		}

		releaseFolderName := "fnf-practice-win64-v" + GIT_TAG_VERSION
		releaseFolder := filepath.Join("release", releaseFolderName)

		if err := MkDir(releaseFolder); err.IsFail {
			return err
		}

		if *FlagDebug {
			WarnLogger.Print("debug flag is ignored for release")
		}
		if *FlagPprof {
			WarnLogger.Print("pprof flag is ignored for release")
		}
		if *FlagDemo {
			WarnLogger.Print("demo flag is ignored for release")
		}

		if err := BuildApp(
			"main.go",
			filepath.Join(releaseFolder, AddExeIfWindows("fnf-practice")),
			false, false, false,
		); err.IsFail {
			return err
		}

		CopyFile("assets/hit-sound.ogg", filepath.Join(releaseFolder, "hit-sound.ogg"), 0664)

		CopyFile("change-hit-sound.txt", filepath.Join(releaseFolder, "change-hit-sound.txt"), 0664)

		// create zip folder
		if err := MkDir("release/zip"); err.IsFail {
			return err
		}

		// zip it
		if err := ZipFile(
			releaseFolder, filepath.Join("release/zip", releaseFolderName+".zip"),
			0664,
		); err.IsFail {

			return err
		}

		return ErrorOk
	}

	switch target {
	case "":
		if err := buildMain(); err.IsFail {
			CrashOnError(err)
		}
	case "font-gen":
		if err := buildFontGen(); err.IsFail {
			CrashOnError(err)
		}
	case "all":
		if err := buildMain(); err.IsFail {
			CrashOnError(err)
		}
		if err := buildFontGen(); err.IsFail {
			CrashOnError(err)
		}
		if err := buildRelease(); err.IsFail {
			CrashOnError(err)
		}
	case "release":
		if err := buildRelease(); err.IsFail {
			CrashOnError(err)
		}
	default:
		ErrLogger.Printf("invalid target %s", target)
		CrashOnError(ErrorDefault)
	}

	Logger.Print("SUCCESS!!")
}

func GetGitTagVersion() (string, SimpleError) {
	cmd := exec.Command(
		"git", "describe", "--tags", "--always", "--abbrev=0",
	)
	cmd.Stderr = os.Stderr

	var gitTagBytes []byte
	var err error
	if gitTagBytes, err = cmd.Output(); err != nil {
		return "", SimpleErrorFromError(err)
	}

	// check if string is valid utf8
	var gitTag string
	if valid := utf8.Valid(gitTagBytes); !valid {
		ErrLogger.Print("git tag string is not valid utf8")
		return "", ErrorDefault
	}

	gitTag = strings.TrimSpace(string(gitTagBytes))

	if strings.ContainsFunc(gitTag, func(r rune) bool { return unicode.IsSpace(r) }) {
		ErrLogger.Print("git tag contains white space")
		return "", ErrorDefault
	}

	return gitTag, SimpleError{}
}

func CrashOnError(err SimpleError) {
	os.Exit(err.ExitCode)
}

func AddExeIfWindows(str string) string {
	if runtime.GOOS == "windows" {
		str += ".exe"
	}

	return str
}

func CopyFile(src, dst string, perm os.FileMode) SimpleError {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	Logger.Printf("copying \"%s\" to \"%s\"", src, dst)
	var err error
	var srcFile []byte
	if srcFile, err = os.ReadFile(src); err != nil {
		ErrLogger.Printf("failed to read %s: %v", src, err)
		return SimpleErrorFromError(err)
	}

	if err = os.WriteFile(dst, srcFile, perm); err != nil {
		ErrLogger.Printf("failed to create %s: %v", dst, err)
		return SimpleErrorFromError(err)
	}

	return SimpleError{}
}

// permission is set to 0755
func MkDir(path string) SimpleError {
	path = filepath.Clean(path)

	Logger.Printf("creating %s folder", path)

	if err := os.MkdirAll(path, 0755); err != nil {
		ErrLogger.Printf("failed to create \"%s\" folder : %v", path, err)
		return SimpleErrorFromError(err)
	}

	return SimpleError{}
}

func BuildApp(
	src, dst string,
	debug, pprof, demo bool,
) SimpleError {
	tags := "noaudio"

	if demo {
		tags += ",demoreplay"
	}

	if pprof {
		tags += ",pprof"
	}

	gcFlags := "-e"

	if debug {
		gcFlags += " -l -N"
	}

	cmd := exec.Command(
		"go",
		"build",
		"-o", dst,
		"-tags="+tags,
		"-gcflags=all="+gcFlags,
		src,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	Logger.Printf("building \"%s\"", dst)
	Logger.Printf("source \"%s\"", src)
	Logger.Printf("command : %s", strings.Join(cmd.Args, " "))

	if err := cmd.Run(); err != nil {
		return SimpleErrorFromError(err)
	}

	return SimpleError{}
}

func zipFileImpl(src string, dst string, perm os.FileMode) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	var err error

	var fileInfo os.FileInfo
	if fileInfo, err = os.Stat(src); err != nil {
		return err
	}

	zipBuffer := new(bytes.Buffer)
	writer := zip.NewWriter(zipBuffer)
	isWriterClosed := false
	defer func() {
		if !isWriterClosed {
			isWriterClosed = true
			writer.Close()
		}
	}()

	if fileInfo.IsDir() {
		srcFS := os.DirFS(src)

		if err = writer.AddFS(srcFS); err != nil {
			return err
		}
	} else {
		if !fileInfo.Mode().IsRegular() {
			return fmt.Errorf("%s in not a regular file", src)
		}

		var srcFile *os.File
		if srcFile, err = os.Open(src); err != nil {
			return err
		}
		defer srcFile.Close()

		_, srcFileName := filepath.Split(src)
		var fw io.Writer
		if fw, err = writer.Create(srcFileName); err != nil {
			return err
		}
		if _, err = io.Copy(fw, srcFile); err != nil {
			return err
		}
	}
	writer.Close()
	isWriterClosed = true

	if err = os.WriteFile(dst, zipBuffer.Bytes(), perm); err != nil {
		return err
	}

	return nil
}

func ZipFile(src string, dst string, perm os.FileMode) SimpleError {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	Logger.Printf("zipping \"%s\" to \"%s\"", src, dst)

	if err := zipFileImpl(src, dst, perm); err != nil {
		ErrLogger.Printf("failed to zip \"%s\": %v", src, err)
		return SimpleErrorFromError(err)
	}

	return ErrorOk
}
