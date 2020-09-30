# photo-organizer
Organize pictures, movies, files in subdirectories

Typical command for a really disorganized folder struct that creates new folder struct in `../Archive` from files in current working working directory. Moves jpegs, and movie files:
```shell script
reco -o ../Archive -d . --keepfolder -t "*.jpg,*.jpeg,*.mov,*.JPG,*.JPEG,*.MOV,*.mp4,*.MP4"
```

run reco.exe -h to see this text:
```
reco is a tool to organize recovered photos super easy.
reco only moves files to a directory. It does not copy nor modify files.
Usages:
        reco [flags]
Example:
        reco -r=false -d ./unorganizedPhotos --year
Flags:
              --actions   Filename to write actions performed for a wet run. CSV format: "Previous location", "New location". (default reco.csv)
              --dimensionMin   Dimension minimum for jpeg/jpg files. (applies to width and height (default 300)
        -d,  --dir   Directory in which to search for files (default )
              --dry   Dry run does nothing (does not move files to directories but still errors or shows verbose output) (default false)
        -t,  --ext   Matching shell file pattern. Separate patterns with commas. See go's filepath.Match() (default *.jpg,*.jpeg,*.mov)
        -h,  --help   Call for help! (default false)
        -k,  --keepfolder   Keep base folder name of file when moving file. Automatically avoids duplicate names such as '/2011/2011/a.jpg' (default false)
        -m,  --month   Organize files by month (month directory) (default false)
              --noerrstop   Do not interrupt file moving due to non-fatal errors (default false)
        -o,  --output   Directory in which to organize files to (default ./recovered)
        -r,  --recursive   Search for files in subdirectories (default true)
              --size   Minimum filesize in MB (default 0)
              --sizeMin   Minimum number of pixels in an image to be processed(jpeg/jpg). Divide this number by a million to get Megapixels. (default 100000)
        -V,  --verbose   Log level, the higher the more verbose.
        Errors:1, Info:2, Print:3, Debug:4 (default 2)
        -y,  --year   Organize files by year (year directory) (default true)
```

Here's a longer extension example:

```bash
-t "*.BMP,*.bmp,*.ppt,*.pptx,*.gif,*.docx,*.DOCX,*.doc,*.DOC,*.3gp,*.3GP,*.MTS,*.mts,*.jpg,*.JPG,*.JPEG,*.jpeg,*.MOV,*.mov,*.mp4,*.MP4,*.wmv,*.WMV,*.mpg,*.MPG,*.mpeg,*.MPEG"
```

and one with more image formats and case insensitive search

```bash
-i -t "*.jp0,*.jpf,*.nef,*.bmp,*.ppt,*.pptx,*.gif,*.docx,*.doc,*.3gp,*.mts,*.jpg,*.jpeg,*.mov,*.mp4,*.wmv,*.mpg,*.mpeg"
```
