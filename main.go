/*
satrace - SAtrace project's CLI tool
Convert formatted text to Date + Data rows asynchronously

Usage:
```
# Dump table
# Dump txt to SAtrace format data
$ satrace table *.txt
2019-8-29 22:23:47  -35   -39.4   -55   ...
2019-8-29 23:34:56  -31   -42.4   -43   ...

# Electric Energy converter
# X axis as line number
$ satrace elen -f 425-575 *.txt

# Electric Energy converter
# X axis as point read from first line configure
$ satrace elen -F 42-46 *.txt

# Peak search
$ satrace peak *.txt
```
*/
package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/mitchellh/cli"
)

const (
	// VERSION info
	VERSION = "v0.1.0"
)

var (
	// version flag
	showVersion bool
	// field code
	field arrayField
	// usecol is column of using calculation
	usecol int
	// logger print to stdout
	format string
	// logger print to stdout
	delim string
	// debug mode
	debug bool
	// wg wait goroutine
	wg sync.WaitGroup
	// logger print to stdout
	logger = log.New(os.Stdout, "", 0)
)

type (
	// arrayField created so that multiple inputs can be accecpted
	arrayField []string
	// configMap is a first line of data
	configMap map[string]string
	// contentArray read from data
	contentArray []float64
	// OutRow is a output line
	OutRow struct {
		Filename string
		Datetime string
		Center   string
		Fields   []float64
	}
	// Command is a list of subcommand
	Command interface {
		// Convert formatted text to Date + Data rows asynchronously
		Help() string
		Table(args []string) int
		Elen(args []string) int
	}
)

func main() {
	// flag.BoolVar(&showVersion, "v", false, "Show version")
	// flag.Var(&field, "f", "Field range such as -f 50-100")
	// flag.IntVar(&usecol, "c", 1, "Column of using calculation")
	// flag.StringVar(&delim, "d", "\t", "A character of separated delimeter")
	// flag.BoolVar(&debug, "debug", false, "Debug mode")
	// flag.Parse()
	// if showVersion {
	// 	fmt.Println("satrace version:", VERSION)
	// 	return // Exit with version info
	// }
	//
	c := cli.NewCLI("satrace", "0.1.0") // subcommand struct
	c.Args = os.Args[1:]
	// Subcommands register
	c.Commands = map[string]cli.CommandFactory{
		"table": func() (cli.Command, error) {
			return &TableCommand{}, nil
		},
		"elen": func() (cli.Command, error) {
			return &ElenCommand{}, nil
		},
	}

	exitCode, err := c.Run()
	if err != nil {
		fmt.Printf("Failed to execute: %s\n", err.Error())
	}
	os.Exit(exitCode)
}

/* Table subcommand */

// TableCommand command definition
type TableCommand struct{}

// Synopsis message of `satrace table`
func (e *TableCommand) Synopsis() string {
	return "satrace subcommand table"
}

// Help message of `satrace table`
func (e *TableCommand) Help() string {
	return "usage: satrace table data/*.txt"
}

// Run print result of writeOutRow()
func (e *TableCommand) Run(args []string) int {
	flags := flag.NewFlagSet("table", flag.ContinueOnError)
	flags.Var(&field, "f", "Field range such as -f 50-100")
	flags.IntVar(&usecol, "c", 1, "Column of using calculation")
	flags.StringVar(&format, "format", "%f", `Print format %f, %e, %E, ...)`)
	flags.BoolVar(&debug, "debug", false, "Debug mode")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	for _, filename := range flags.Args() {
		// File not exist then next loop so that filtering here
		// flags.Args() contains all flag and filename args
		if _, err := os.Stat(filename); err != nil {
			continue
		}
		wg.Add(1)
		go func(f string) {
			defer wg.Done()
			var err error
			o, err := e.writeOutRow(f)
			if err != nil {
				panic(err)
			}
			logger.Println(o)
		}(filename)
	}
	wg.Wait()
	return 0
}

// writeOutRow return a line of processed content
func (e *TableCommand) writeOutRow(s string) (o OutRow, err error) {
	var (
		config  configMap
		content contentArray
		m, n    int
	)
	o.Filename = s
	o.Datetime = parseDatetime(filepath.Base(s))
	config, content, err = readTrace(s)
	if err != nil {
		return
	}
	if debug {
		logger.Printf("[ CONFIG ]:%v\n", config)
		logger.Printf("[ CONTENT ]:%v\n", content)
		logger.Printf("[ FIELD ]:%v\n", field)
	}
	o.Center = config[":FREQ:CENT"]
	if len(field) > 0 { // => arrayField{} : [["50-100"] ["300-350"]...]
		for _, f := range field {
			m, n, err = parseField(f) // => [[50 100] [300 350]...]
			if err != nil {
				return
			}
			for _, mw := range content[m : n+1] {
				o.Fields = append(o.Fields, mw)
			}
		}
	} else { // no -f flag
		o.Fields = content
	}
	// Debug print format
	if debug {
		logger.Printf("[ TYPE OUTROW ]%v\n", o)
		// continue // print not standard output
		return
	}
	return
}

/* Elen subcommand */

// ElenCommand command definition
type ElenCommand struct{}

// Synopsis message of `satrace elen`
func (e *ElenCommand) Synopsis() string {
	return "satrace subcommand elen"
}

// Help message of `satrace elen`
func (e *ElenCommand) Help() string {
	return "usage: satrace elen -f 50-100"
}

// Run print result of writeOutRow()
func (e *ElenCommand) Run(args []string) int {
	flags := flag.NewFlagSet("elen", flag.ContinueOnError)
	flags.Var(&field, "f", "Field range such as -f 50-100")
	flags.IntVar(&usecol, "c", 1, "Column of using calculation")
	flags.StringVar(&format, "format", "%f", `Print format %f, %e, %E, ...)`)
	flags.BoolVar(&debug, "debug", false, "Debug mode")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	for _, filename := range flags.Args() {
		// File not exist then next loop so that filtering here
		// flags.Args() contains all flag and filename args
		if _, err := os.Stat(filename); err != nil {
			continue
		}
		wg.Add(1)
		go func(f string) {
			defer wg.Done()
			var err error
			o, err := e.writeOutRow(f)
			if err != nil {
				panic(err)
			}
			logger.Println(o)
		}(filename)
	}
	wg.Wait()
	return 0
}

// writeOutRow return a line of processed content
func (e *ElenCommand) writeOutRow(s string) (o OutRow, err error) {
	var (
		config  configMap
		content contentArray
		m, n    int
	)
	o.Filename = s
	o.Datetime = parseDatetime(filepath.Base(s))
	config, content, err = readTrace(s)
	if err != nil {
		return
	}
	if debug {
		logger.Printf("[ CONFIG ]:%v\n", config)
		logger.Printf("[ CONTENT ]:%v\n", content)
		logger.Printf("[ FIELD ]:%v\n", field)
	}
	o.Center = config[":FREQ:CENT"]
	if len(field) > 0 { // => arrayField{} : [["50-100"] ["300-350"]...]
		for _, f := range field {
			m, n, err = parseField(f)
			if err != nil {
				return
			}
			mw := content.signalBand(m, n)
			o.Fields = append(o.Fields, mw)
		}
	} else { // no -f flag
		var end int
		end, err = strconv.Atoi(config[":SWE:POIN"])
		if err != nil {
			return
		}
		o.Fields = []float64{content.signalBand(0, end-1)}
	}
	// Debug print format
	if debug {
		logger.Printf("[ TYPE OUTROW ]%v\n", o)
		// continue // print not standard output
		return
	}
	return
}

// OutRow.String print as comma separated value
func (o OutRow) String() string {
	s := fmt.Sprintf("%s,%s,%s", // comma separated
		o.Datetime,
		o.Center,
		strings.Join(func() (ss []string) {
			for _, f := range o.Fields { // convert []float64=>[]string
				// s := strconv.FormatFloat(f, 'f', -1, 64)
				s := fmt.Sprintf(format, f)
				ss = append(ss, s)
			}
			return
		}(), ","), // comma separated
	)
	return s
}

// signalBand convert mWatt then sum between band
func (c contentArray) signalBand(m, n int) (mw float64) {
	for i := m; i <= n; i++ {
		mw += db2mw(c[i])
	}
	return
}

// parseConfig convert first line of data to config map
func parseConfig(b []byte) configMap {
	config := make(configMap)
	sarray := bytes.Split(b, []byte(";"))
	// snip # 20200627_180505 *RST & *CLS
	// chomp last new line
	sa := sarray[2 : len(sarray)-1]
	for _, e := range sa {
		kv := strings.Fields(string(e))
		config[kv[0]] = strings.Join(kv[1:], " ")
	}
	return config
}

// parseFilename convert a filename as datetime (%Y-%m-%d %H:%M:%S) format
func parseDatetime(s string) string {
	return fmt.Sprintf("%s-%s-%s %s:%s:%s", // 2006-01-02 15:05:12
		s[0:4], s[4:6], s[6:8], s[9:11], s[11:13], s[13:15])
}

// parseField convert -f option to 2 int pair
func parseField(s string) (i0, i1 int, err error) {
	if !strings.Contains(s, "-") {
		err = errors.New("Error: Field flag -f " + s +
			" not contains range \"-\", use int-int")
		return
	}
	ss := strings.Split(s, "-")
	i0, err = strconv.Atoi(ss[0])
	i1, err = strconv.Atoi(ss[1])
	if i0 > i1 {
		err = fmt.Errorf("Error: Must be lower %d than %d", i0, i1)
	}
	return
}

// arrayField.String sets multiple -f flag
func (i *arrayField) String() string {
	// change this, this is just can example to satisfy the interface
	return "my string representation"
}

// arrayField.Set sets multiple -f flag
func (i *arrayField) Set(value string) error {
	*i = append(*i, strings.TrimSpace(value))
	return nil
}

// db2mw returns dB convert to mWatt
func db2mw(db float64) float64 {
	return math.Pow(10, db/10)
}

// readTrace read from a filename to `config` from first line,
// `content` from no # line.
func readTrace(filename string) (config configMap, content contentArray, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var (
		line   []byte
		isConf = true
		f      float64
	)
	for {
		line, _, err = reader.ReadLine()
		if isConf { // First line is configure
			config = parseConfig(line)
			isConf = false
			continue
		}
		if bytes.HasPrefix(line, []byte("#")) {
			// Got "# <eof>" successful terminationthen
			return
		}
		if err == io.EOF { // if EOF then finish func
			err = nil
			logger.Println("warning: data has not <eof>")
			return // might not work because HasPrefix([]byte("#"))
		}
		if err != nil { // if error at ReadLine then finish func
			return
		}
		// Trim Prefix/Surfix/Middle whitespace
		bb := bytes.Fields(bytes.TrimSpace(line))
		f, err = strconv.ParseFloat(string(bb[usecol]), 64)
		if err != nil {
			return
		}
		content = append(content, f)
	}
}
