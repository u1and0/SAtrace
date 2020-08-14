/*
satrace - SAtrace project's CLI tool

Convert formatted text to Data rows as CSV asynchronously.

Usage:

1. Dump table
Dump txt to SAtrace format data, use `table` subcommand.

```
$ satrace table -f 100-200 *.txt
2019-8-29 22:23:47  -35   -39.4   -55   ...
2019-8-29 23:34:56  -31   -42.4   -43   ...
```

2. Electric Energy converter, use `elen` subcommand
Sum specified line of antilogarithm data content.
`elen` is abbreviation of "ELectric ENergy".

```
$ satrace elen -f 425-575 *.txt
```


3. Peak search, use `peak` subcommand
Extract frequency of peak which value is larger than delta by Noise Floor.
Noise Floor is defined first quantile.

```
$ satrace peak -d 10 *.txt
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
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"

	"github.com/mitchellh/cli"
	"github.com/montanaflynn/stats"
)

var (
	// field code
	field arrayField
	// usecol is column of using calculation
	usecol int
	// format is display format like %f, %e, %E
	format string
	// delim is character of delimiter
	delim string
	// show is string of format of columns
	show string
	// delta use peak search value lower by delta
	delta float64
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

	// OutRow is a output line
	OutRow struct {
		Filename   string
		Datetime   string
		Center     string
		Fields     []float64
		Format     string
		NoiseFloor float64
		Show       string
	}
	// Command is a list of subcommand
	Command interface {
		// Convert formatted text to Date + Data rows asynchronously
		Help() string
		Table(args []string) int
		Elen(args []string) int
	}
	// T is struct of yaml config
	T struct {
		Subcommand string `yaml:"subcommand"`
		Options    struct {
			Field  arrayField `yaml:"field"`
			C      string     `yaml:"c"`
			Format string     `yaml:"format"`
			Show   string     `yaml:"show"`
			D      string     `yaml:"d"`
			Debug  string     `yaml:"debug"`
		}
		Output string `yaml:"output"`
	}
)

func main() {
	c := cli.NewCLI("satrace", "0.2.0")
	if len(os.Args) < 2 { // no argument
		raw, err := ioutil.ReadFile("./config.yml")
		if err != nil {
			logger.Fatalf("%s", err.Error())
			os.Exit(1)
		}
		t := T{}
		err = yaml.Unmarshal(raw, &t)
		if err != nil {
			logger.Fatalf("%s", err.Error())
			os.Exit(1)
		}
		ss := []string{
			t.Subcommand,
			t.Options.C,
			t.Options.Format,
			t.Options.Show,
			t.Options.D,
			t.Options.Debug,
		}
		c.Args = append(ss, t.Options.Field...)
	} else { // has subcommand and options
		c.Args = os.Args[1:]
	}
	// Subcommands register
	c.Commands = map[string]cli.CommandFactory{
		"table": func() (cli.Command, error) {
			return &TableCommand{}, nil
		},
		"elen": func() (cli.Command, error) {
			return &ElenCommand{}, nil
		},
		"peak": func() (cli.Command, error) {
			return &PeakCommand{}, nil
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
	return "Extract data column to row. Returns dB of text field."
}

// Help message of `satrace table`
func (e *TableCommand) Help() string {
	return "usage: satrace table -f 100-200 -c 2 data/*.txt"
}

// Run print result of writeOutRow()
func (e *TableCommand) Run(args []string) int {
	flags := flag.NewFlagSet("table", flag.ContinueOnError)
	flags.Var(&field, "f", "Field range such as -f 50-100")
	flags.IntVar(&usecol, "c", 1, "Column of using calculation")
	flags.StringVar(&format, "format", "%f", `Print format %f, %e, %E, ...)`)
	flags.StringVar(&show, "show", "date,center,noise", "Print columns separated comma")
	flags.BoolVar(&debug, "debug", false, "Debug mode")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	// Add header
	logger.Printf("%s,%s", show, func() string { return strings.Join(field, ",") }())
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
		df   Trace
		m, n int
	)
	o.Filename = s
	o.Format = format
	o.Datetime = parseDatetime(filepath.Base(s))
	o.Show = show
	df, err = readTrace(s, usecol)
	if err != nil {
		return
	}
	if debug {
		logger.Printf("[ CONFIG ]:%v\n", df.Config)
		logger.Printf("[ CONTENT ]:%v\n", df.Content)
		logger.Printf("[ FIELD ]:%v\n", field)
	}
	o.Center = df.Config[":FREQ:CENT"]
	o.NoiseFloor = df.noisefloor()
	if len(field) > 0 { // => arrayField{} : [["50-100"] ["300-350"]...]
		for _, f := range field {
			m, n, err = parseField(f) // => [[50 100] [300 350]...]
			if err != nil {
				return
			}
			for _, mw := range df.Content[m : n+1] {
				o.Fields = append(o.Fields, mw)
			}
		}
	} else { // no -f flag
		o.Fields = df.Content
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
	return "Electric Energy converter. Returns millWatt of field sum."
}

// Help message of `satrace elen`
func (e *ElenCommand) Help() string {
	return "usage: satrace elen -f 50-100 --format %e trace/*.txt"
}

// Run print result of writeOutRow()
func (e *ElenCommand) Run(args []string) int {
	flags := flag.NewFlagSet("elen", flag.ContinueOnError)
	flags.Var(&field, "f", "Field range such as -f 50-100")
	flags.IntVar(&usecol, "c", 1, "Column of using calculation")
	flags.StringVar(&format, "format", "%f", `Print format %f, %e, %E, ...)`)
	flags.StringVar(&show, "show", "date,center,noise", "Print columns separated comma")
	flags.BoolVar(&debug, "debug", false, "Debug mode")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	// Add header
	logger.Printf("%s,%s", show, func() string { return strings.Join(field, ",") }())
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
		df   Trace
		m, n int
	)
	o.Filename = s
	o.Format = format
	o.Datetime = parseDatetime(filepath.Base(s))
	o.Show = show
	df, err = readTrace(s, usecol)
	if err != nil {
		return
	}
	if debug {
		logger.Printf("[ CONFIG ]:%v\n", df.Config)
		logger.Printf("[ CONTENT ]:%v\n", df.Content)
		logger.Printf("[ FIELD ]:%v\n", field)
	}
	o.Center = df.Config[":FREQ:CENT"]
	o.NoiseFloor = df.noisefloor()
	if len(field) > 0 { // => arrayField{} : [["50-100"] ["300-350"]...]
		for _, f := range field {
			m, n, err = parseField(f)
			if err != nil {
				return
			}
			mw := df.signalBand(m, n)
			o.Fields = append(o.Fields, mw)
		}
	} else { // no -f flag
		var end int
		end, err = strconv.Atoi(df.Config[":SWE:POIN"])
		if err != nil {
			return
		}
		o.Fields = []float64{df.signalBand(0, end-1)}
	}
	// Debug print format
	if debug {
		logger.Printf("[ TYPE OUTROW ]%v\n", o)
		// continue // print not standard output
		return
	}
	return
}

/* Peak search subcommand */

// PeakCommand command definition
type PeakCommand struct{}

// Synopsis message of `satrace peak`
func (e *PeakCommand) Synopsis() string {
	return "Peak search method. Returns frequency of peak which value is larger than delta."
}

// Help message of `satrace peak`
func (e *PeakCommand) Help() string {
	return "usage: satrace peak -f 50-100 -d 10 -c 1 --format %.3f trace/*.txt"
}

// Run print result of writeOutRow()
func (e *PeakCommand) Run(args []string) int {
	flags := flag.NewFlagSet("peak", flag.ContinueOnError)
	// flags.Var(&field, "f", "Field range such as -f 50-100")
	flags.IntVar(&usecol, "c", 1, "Column of using calculation")
	flags.StringVar(&format, "format", "%f", `Print format (%f, %.3f, %e, %E...)`)
	flags.Float64Var(&delta, "d", 1, "Use peak search value lower by delta")
	flags.StringVar(&show, "show", "date,center,noise", "Print columns separated comma")
	flags.BoolVar(&debug, "debug", false, "Debug mode")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	// Add header
	logger.Printf("%s,%s", show, func() string { return strings.Join(field, ",") }())
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
func (e *PeakCommand) writeOutRow(s string) (o OutRow, err error) {
	var (
		df Trace
		v  []float64
	)
	o.Filename = s
	o.Format = format
	o.Datetime = parseDatetime(filepath.Base(s))
	o.Show = show
	df, err = readTrace(s, usecol)
	if err != nil {
		return
	}
	o.Center = df.Config[":FREQ:CENT"]
	o.NoiseFloor = df.noisefloor()
	o.Fields, v = df.peakSearch(delta)
	// Debug print format
	if debug {
		logger.Printf("[ CONFIG ]:%v\n", df.Config)
		logger.Printf("[ CONTENT ]:%v\n", df.Content)
		logger.Printf("[ FIELD ]:%v\n", field)
		logger.Printf("[ TYPE OUTROW ]%v\n", o)
		logger.Printf("[ INDEX OF PEAK ]%v\n", o.Fields)
		logger.Printf("[ VALUE OF PEAK ]%v\n", v)
		// continue // print not standard output
	}
	return
}

// stringField join comma separated filed values
func (o OutRow) stringField() string {
	var ss []string
	for _, f := range o.Fields { // convert []float64=>[]string
		s := fmt.Sprintf(o.Format, f)
		ss = append(ss, s)
	}
	return strings.Join(ss, ",") // comma separated
}

// stringField join comma separated filed values
func (o OutRow) stringShows() string {
	var ss []string
	for _, s := range strings.Split(o.Show, ",") {
		switch s {
		case "date":
			ss = append(ss, o.Datetime)
		case "center":
			ss = append(ss, o.Center)
		case "noise":
			ss = append(ss, fmt.Sprintf(o.Format, o.NoiseFloor))
		}
	}
	return strings.Join(ss, ",") // comma separated
}

// OutRow.String print as comma separated value
func (o OutRow) String() string {
	return fmt.Sprintf("%s,%s", o.stringShows(), o.stringField())
}

// peakSearch search values of local maxima (peaks)
func (c Trace) peakSearch(delta float64) (pi, pe []float64) {
	nf := c.noisefloor()
	for i, e := range c.Content {
		if e-nf > delta {
			pi = append(pi, c.Index[i])
			pe = append(pe, e)
		}
	}
	return
}

// noisefloor define as first quantile
func (c Trace) noisefloor() float64 {
	const QUANTILE = 25
	nf, err := stats.Percentile(c.Content, QUANTILE)
	if err != nil {
		logger.Printf("error %s", err)
	}
	return nf
}

func parseIndex(c configMap) []float64 {
	center := asFloat64(c[":FREQ:CENT"])
	span := asFloat64(c[":FREQ:SPAN"])
	points := int(asFloat64(c[":SWE:POIN"]))
	starts := center - span/2
	finish := center + span/2
	div := (finish - starts) / float64(points-1)
	index := make([]float64, points)
	for i := 0; i < points; i++ {
		index[i] = starts
		starts += div
	}
	return index
}

func asFloat64(s string) float64 {
	f, err := strconv.ParseFloat(strings.Fields(s)[0], 64)
	if err != nil {
		panic(err)
	}
	return f
}

// signalBand convert mWatt then sum between band
func (c Trace) signalBand(m, n int) (mw float64) {
	for i := m; i <= n; i++ {
		mw += db2mw(c.Content[i])
	}
	return
}

// parseConfig convert first line of data to config map
func parseConfig(b []byte) configMap {
	// CHOMP snip # 20200627_180505 *RST & *CLS
	const CHOMP = 2
	config := make(configMap)
	sarray := bytes.Split(b, []byte(";"))
	sa := sarray[CHOMP : len(sarray)-1] // chomp last new line
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

// Trace is a set of config & data column read from a txt file
type Trace struct {
	// Config is a first line of data
	Config map[string]string
	// Content read from data
	Content []float64
	// Index is parse from config center and config point
	Index []float64
	// Unit is a index unit kHz, MHz, GHz... read from config
	Unit string
}

// readTrace read from a filename to `config` from first line,
// `content` from no # line.
func readTrace(filename string, usecol int) (df Trace, err error) {
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
			config := parseConfig(line)
			df.Config = config
			df.Index = parseIndex(config)
			df.Unit = strings.Fields(config[":FREQ:CENT"])[1]
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
		df.Content = append(df.Content, f)
	}
}
