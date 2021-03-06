// dun_dump is a tool for constructing dungeons, based on the information
// retrieved from a given DUN file, and storing these dungeons as png images.
//
// Usage:
//
//    dun_dump [OPTION]... [name.dun]...
//
// Flags:
//
//    -a=false
//            Dump all dungeons.
//    -celini="cel.ini"
//            Path to an ini file containing image information.
//            Note: 'cl2.ini' will be used for files that have the '.cl2' extension.
//    -mpqdump="mpqdump/"
//            Path to an extracted MPQ file.
//    -mpqini="mpq.ini"
//            Path to an ini file containing relative path information.
package main

import (
	"flag"
	dbg "fmt"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/mewkiz/pkg/imgutil"
	"github.com/mewrnd/blizzconv/configs/dun"
	"github.com/mewrnd/blizzconv/configs/dunconf"
	"github.com/mewrnd/blizzconv/configs/min"
	"github.com/mewrnd/blizzconv/images/cel"
	"github.com/mewrnd/blizzconv/images/imgconf"
	"github.com/mewrnd/blizzconv/mpq"
)

var flagAll bool

func init() {
	flag.Usage = usage
	flag.BoolVar(&flagAll, "a", false, "Dump all dungeons.")
	flag.StringVar(&imgconf.IniPath, "celini", "cel.ini", "Path to an ini file containing image information.")
	flag.StringVar(&dunconf.IniPath, "dunini", "dun.ini", "Path to an ini file containing starting coordinate information.")
	flag.StringVar(&mpq.ExtractPath, "mpqdump", "mpqdump/", "Path to an extracted MPQ file.")
	flag.StringVar(&mpq.IniPath, "mpqini", "mpq.ini", "Path to an ini file containing relative path information.")
	flag.Parse()
	err := mpq.Init()
	if err != nil {
		log.Fatalln(err)
	}
	err = dunconf.Init()
	if err != nil {
		log.Fatalln(err)
	}
	err = imgconf.Init()
	if err != nil {
		log.Fatalln(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTION]... [name]...\n", os.Args[0])
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Flags:")
	flag.PrintDefaults()
}

func main() {
	var dungeonNames []string
	if flagAll {
		dungeonNames = dunconf.DungeonNames()
	} else if flag.NArg() > 0 {
		dungeonNames = flag.Args()
	} else {
		flag.Usage()
		os.Exit(1)
	}
	for _, dungeonName := range dungeonNames {
		err := dungeonDump(dungeonName)
		if err != nil {
			log.Println(err)
		}
	}
}

// dumpPrefix is the name of the dump directory.
const dumpPrefix = "_dump_/"

// dungeonDump creates a dump directory and stores the dungeon, which has been
// constructed based on the given DUN files, as a png image once for each image
// config (pal).
func dungeonDump(dungeonName string) (err error) {
	dunNames, err := dunconf.GetDunNames(dungeonName)
	if err != nil {
		return err
	}
	dungeon := dun.New()
	for _, dunName := range dunNames {
		err = dungeon.Parse(dunName)
		if err != nil {
			return fmt.Errorf("failed to parse %q: %s", dungeonName, err)
		}
	}
	colCount, err := dunconf.GetColCount(dungeonName)
	if err != nil {
		return err
	}
	rowCount, err := dunconf.GetRowCount(dungeonName)
	if err != nil {
		return err
	}
	nameWithoutExt, err := dun.GetLevelName(dunNames[0])
	if err != nil {
		return err
	}
	minName := nameWithoutExt + ".min"
	pillars, err := min.Parse(minName)
	if err != nil {
		return err
	}
	imgName := nameWithoutExt + ".cel"
	relPalPaths := imgconf.GetRelPalPaths(imgName)
	for _, relPalPath := range relPalPaths {
		conf, err := cel.GetConf(imgName, relPalPath)
		if err != nil {
			return err
		}
		var palDir string
		if len(relPalPaths) > 1 {
			dbg.Println("using pal:", relPalPath)
			palDir = dungeonName + "/"
		}
		levelFrames, err := cel.DecodeAll(imgName, conf)
		if err != nil {
			return err
		}
		dumpDir := path.Clean(dumpPrefix+"_dungeons_/") + "/" + palDir
		// prevent directory traversal
		if !strings.HasPrefix(dumpDir, dumpPrefix) {
			return fmt.Errorf("path (%s) contains no dump prefix (%s).", dumpDir, dumpPrefix)
		}
		err = os.MkdirAll(dumpDir, 0755)
		if err != nil {
			return err
		}
		dungeonPath := dumpDir + dungeonName + ".png"
		if len(relPalPaths) > 1 {
			palName := path.Base(relPalPath)
			palNameWithoutExt := palName[:len(palName)-len(path.Ext(palName))]
			dungeonPath = dumpDir + dungeonName + "_" + palNameWithoutExt + ".png"
		}
		dbg.Println("Creating image:", path.Base(dungeonPath))
		img := dungeon.Image(colCount, rowCount, pillars, levelFrames)
		err = imgutil.WriteFile(dungeonPath, img)
		if err != nil {
			return err
		}
	}
	return nil
}
