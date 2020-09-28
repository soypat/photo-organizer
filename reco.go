package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"golang.org/x/image/bmp"
)

const helpMsg = `reco is a tool to organize recovered photos super easy.
reco only moves files to a directory. It does not copy nor modify files.
reco can decode jpeg/png/bmp files to apply size filters.
Usages:
	reco [flags]
Example:
	reco -r=false -d ./unorganizedPhotos --month
Flags:`

// MBtoBytes converts MB to bytes by simple multiplication
const MBtoBytes = 1000 * 1000

const (
	defaultRecoveryDir = "./recovered"
)

var ( // flags
	dir, exts, saveDir, actionFile                                       string
	mflag, yflag, recursive, dry, help, verbose, interactive, keepFolder bool
	ignoreFileErr                                                        bool
	minDim, logLevel, dimensionMin, sizePixelMin, sizeMin                int
	actionFp                                                             *os.File
)

func init() {
	debugf("parsing flags")
	pflag.StringVarP(&dir, "dir", "d", "", "Directory in which to search for files")
	pflag.StringVarP(&saveDir, "output", "o", defaultRecoveryDir, "Directory in which to organize files to")
	pflag.StringVarP(&exts, "ext", "t", "*.jpg,*.jpeg,*.mov", "Matching shell file pattern. Separate patterns with commas. See go's filepath.Match()")
	pflag.BoolVarP(&recursive, "recursive", "r", true, "Search for files in subdirectories")
	//
	pflag.StringVar(&actionFile, "actions", "reco.csv", "Filename to write actions performed for a wet run. CSV format: \"Previous location\", \"New location\".")
	pflag.IntVarP(&logLevel, "verbose", "V", 2, "Log level. The higher, the more verbose.\n\t\tErrors:1, Info:2, Print:3, Debug:4")
	pflag.BoolVar(&dry, "dry", false, "Dry run does nothing (does not move files to directories but still errors or shows verbose output)")
	//
	pflag.BoolVar(&ignoreFileErr, "noerrstop", false, "Do not interrupt file moving due to non-fatal errors")
	pflag.BoolVarP(&keepFolder, "keepfolder", "k", false, "Keep base folder name of file when moving file. Automatically avoids duplicate names such as '/2011/2011/a.jpg'")
	// time
	pflag.BoolVarP(&yflag, "year", "y", true, "Organize files by year (year directory)")
	pflag.BoolVarP(&mflag, "month", "m", false, "Organize files by month (month directory)")
	// size of file and pictures
	pflag.IntVar(&dimensionMin, "dimensionMin", 300, "Dimension minimum for files. (applies to width and height")
	pflag.IntVar(&sizeMin, "size", 0, "Minimum filesize in MB")
	pflag.IntVar(&sizePixelMin, "sizeMin", 100000, "Minimum number of pixels in an image to be processed(jpeg/jpg). Divide this number by a million to get Megapixels.")

	pflag.BoolVarP(&help, "help", "h", false, "Call for help!")
	pflag.Lookup("help").Hidden = true
	pflag.Parse()
	if help {
		printHelp()
		os.Exit(0)
	}
	if dir == "" {
		dry, interactive = true, true
		fmt.Print("-d or --dir flag is required, reco will now run in dry mode! Run `reco -h` for help.\nType in desired directory:")
		fmt.Scanln(&dir)
	}
}

func run() error {
	var totalMovedSize int64
	var files []string
	infof("starting reco")
	printf("logLevel: %d, dry:%t", logLevel, dry)
	finfos, _ := ioutil.ReadDir(dir)
	cwd, _ := os.Getwd()
	debugf("directories/files listed in %s: %+v", cwd, finfos)

	err := os.MkdirAll(saveDir, 0600)
	if err != nil {
		printf("error while making dir %s: %s", saveDir, err)
	}
	filecounter := 0
	extensions := strings.Split(exts, ",")
	if len(extensions) == 0 {
		fatalf("no pattern for --ext found")
	}
	if err := os.Mkdir("dir", 0666); os.IsNotExist(err) {
		fatalf("directory  %s not exist", dir)
	}
	if recursive {
		for _, ext := range extensions {
			debugf("looking for %s in dir: %s", ext, dir)
			filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if info == nil {
					errorf("got nil info for %s", path)
					return nil
				}
				if info.IsDir() {
					return nil
				}
				path = strings.ReplaceAll(path, "\\", "/")
				match, err := filepath.Match(ext, path)
				if err != nil {
					fatalf("pattern %s may be malformed, reading %s: %s", ext, path, err)
				}
				if match {
					files = append(files, path)
				}
				return nil
			})
		}
	} else {
		finfos, err := ioutil.ReadDir(dir)
		if err != nil {
			fatalf("reading base directory %s: %s", dir, err)
		}
		for _, finfo := range finfos {
			if finfo.IsDir() {
				continue
			}
			for _, ext := range extensions {
				name := strings.ReplaceAll(finfo.Name(), "\\", "/")
				match, err := filepath.Match(ext, name)
				if err != nil {
					fatalf("pattern %s may be malformed, reading %s: %s", ext, name, err)
				}
				if match {
					files = append(files, filepath.Join(dir, name))
				}
			}
		}
	}
	infof("finished getting %d files", len(files))
	debugf("Files: %v", files)
	if len(files) == 0 {
		fatalf("no files found with patterns %v", extensions)
	}
	if !dry {
		actionFp, err = os.Create(actionFile)
		if err != nil {
			fatalf("could not create actions file %s: %s", actionFile, err)
		}
		defer actionFp.Close()
		defer actionFp.Sync()
	}
	for _, file := range files {
		folder, _ := filepath.Split(file)
		ext := strings.ToLower(filepath.Ext(file)) // file extension
		fp, err := os.Open(file)                   // file pointer
		if err != nil {
			return err
		}
		var subfolder string
		switch ext {
		case ".nef", ".cr2", ".crw", ".erf", ".3fr", ".kdc", ".mos", ".nrw", ".tiff", ".tif":
			subfolder = "photos"
		case ".jpeg", ".jpg", ".png", ".bmp":
			var im image.Config
			if ext == ".bmp" {
				im, err = bmp.DecodeConfig(fp)
			} else {
				im, _, err = image.DecodeConfig(fp)
			}

			if err != nil {
				if ignoreFileErr {
					im = image.Config{Height: 3000, Width: 3000}
				} else {
					errorf("decoding: %s: %s\n", file, err)
					continue
				}
			}
			size := im.Height * im.Width
			if size < sizePixelMin || im.Height < dimensionMin || im.Width < dimensionMin {
				debugf("pixel size of %s too small", file)
				continue
			}
			subfolder = "photos"
		case ".mov", ".3gp", ".mp4", ".mpeg", ".wmv", ".mts", ".avi", ".m4p", ".m4b", ".m4v", ".m4a", ".m4r", ".f4v":
			subfolder = "movies"
		case ".wav", ".mp3":
			subfolder = "audio"
		case ".wmf", ".flv", ".svg", ".ai", ".gif", ".thm":
			subfolder = "media"
		case ".zip":
			subfolder = "zips"
		case ".doc", ".docx", ".ppt", ".pptx", ".xls", ".xlsx":
			subfolder = "docs"
		case ".pdf":
			subfolder = "pdf"
		default:
			subfolder = "other"
		}
		info, err := fp.Stat()
		if err != nil {
			errorf("getting stats for %s: %s", file, err)
			continue
		}
		size := info.Size()
		if size < int64(sizeMin)*MBtoBytes {
			debugf("skipping file %s: %dMB too small", file, size/MBtoBytes)
			continue
		}
		if yflag || mflag {
			if yflag {
				subfolder += fmt.Sprintf("/%d", info.ModTime().Year())
			}
			if mflag {
				subfolder += fmt.Sprintf("/%d", info.ModTime().Month())
			}
		}
		if keepFolder && filepath.Base(subfolder) != filepath.Base(folder) && filepath.Clean(dir) != filepath.Clean(folder) {
			subfolder = filepath.Join(subfolder, filepath.Base(folder))
		}
		err = fp.Close()
		if err != nil {
			fatalf("closing file %s: %s", file, err)
		}

		err = mv(file, filepath.Join(saveDir, subfolder))
		if err != nil {
			if ignoreFileErr {
				errorf("error moving %s -> %s: %s", file, filepath.Join(saveDir, subfolder), err)
			} else {
				fatalf("error moving %s -> %s: %s", file, filepath.Join(saveDir, subfolder), err)
			}
		}
		totalMovedSize += size
		filecounter++
	}

	infof("processed %d files (%s)", filecounter, fmtByte(totalMovedSize))
	if interactive {
		infof("Press enter to end reco.")
		fmt.Scanln(&dir)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	infof("finished reco")
}

func mv(filedir, newdir string) error {
	if dry {
		infof("{dry} not moving %s -> %s", filedir, newdir)
		return nil
	}
	debugf("moving %s -> %s", filedir, newdir)
	_ = os.MkdirAll(newdir, 0600)
	_, file := filepath.Split(filedir)
	newName := filepath.Join(newdir, file)
	err := os.Rename(filedir, newName)
	if err != nil {
		return err
	}
	_, err = os.Stat(newName)
	if err != nil {
		return err
	}
	_, err = actionFp.Write([]byte(fmt.Sprintf("\"%s\",\"%s\"\n", filedir, newName)))
	return err
}

func sliceContains(slis []string, s string) int {
	for i, sli := range slis {
		if sli == s {
			return i
		}
	}
	return -1
}

func debugf(format string, args ...interface{}) {
	if logLevel >= 4 {
		logf("debu", format, args)
	}
}
func printf(format string, args ...interface{}) {
	if logLevel >= 3 {
		logf("prin", format, args)
	}
}
func infof(format string, args ...interface{}) {
	if logLevel >= 2 {
		logf("info", format, args)
	}
}
func errorf(format string, args ...interface{}) {
	if logLevel >= 1 {
		logf("erro", format, args)
	}
}
func fatalf(format string, args ...interface{}) { logf("fata", format, args); os.Exit(1) }
func logf(tag, format string, args []interface{}) {
	msg := fmt.Sprintf(format, args...)
	if args == nil {
		msg = fmt.Sprintf(format)
	}
	fmt.Println(fmt.Sprintf("[%s] %s", tag, msg))
}

func printHelp() {
	fmt.Println(helpMsg)
	pflag.VisitAll(func(flag *pflag.Flag) {
		if flag.Shorthand == "" {
			fmt.Printf("\t      --%s   %s (default %s)\n ", flag.Name, flag.Usage, flag.DefValue)
		} else {
			fmt.Printf("\t-%s,  --%s   %s (default %s)\n ", flag.Shorthand, flag.Name, flag.Usage, flag.DefValue)
		}
	})
}

func fmtByte(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
