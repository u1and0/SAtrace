satracli - SAtrace project's CLI tool

Convert formatted text to Data rows as CSV asynchronously.

# Usage

```
$ satracli SUBCOMMAND [OPTIONS] PATH ...
```

## Subcommand
* table - Extract data column to row. Returns dB of text field.
* elen  - Electric Energy converter. Returns millWatt of field sum.
* peak  - Peak search method. Returns frequency of peak which value is larger than delta.


### Dump table
Dump txt to SAtrace format data, use `table` subcommand.

![tablepng](https://raw.githubusercontent.com/u1and0/satracli/u1and0-patch-1/gosatrace.png)

```
$ satracli table -f 100-200 *.txt
2019-8-29 22:23:47  -35   -39.4   -55   ...
2019-8-29 23:34:56  -31   -42.4   -43   ...
```

### Electric Energy converter, use `elen` subcommand
Sum specified line of antilogarithm data content.

<img src="https://latex.codecogs.com/gif.latex?f(x)=\sum_{i=m}^n10^\frac{x_i}{10}"/>

`elen` is abbreviation of "ELectric ENergy".

```
$ satracli elen -f 425-575 *.txt
```


### Peak search, use `peak` subcommand
Extract frequency of peak which value is larger than delta by Noise Floor.
Noise Floor is defined first quantile.

```
$ satracli peak -d 10 *.txt
```

## Options
### Common options
* -c: Column of using calculation
* --format: Print format %f, %e, %E, ...
* --show: Print columns separated comma
* --debug: Debug mode

### `table` subcommand options
* -f: Filed range as point (multiple OK)

```
$ satracli table -f 0-75 -f 205-280 -f 425-575 -f 725-800 -f 925-1000 *.txt
```



### `elen` subcommand options
* -f: Filed range as point (multiple OK)

```
$ satracli elen -f 0-75 -f 205-280 -f 425-575 -f 725-800 -f 925-1000 *.txt
```


### `peak` subcommand options
* -d: Use peak search value lower by delta


# Data Structure
```
# 20200627_180505 *RST;*CLS;:INP:COUP DC;:BAND:RES 1 Hz;:AVER:COUNT 10;:SWE:POIN 1001;:FREQ:CENT 22.2 kHz;:FREQ:SPAN 2 kHz;:TRAC1:TYPE AVER;:INIT:CONT 0;:FORM REAL,32;:FORM:BORD SWAP;:INIT:IMM;:POW:ATT 0;:DISP:WIND:TRAC:Y:RLEV -30 dBm;
   0     -93.21
   1     -93.97
   2     -94.93
   3     -84.87
   4     -96.31
   5     -95.23
   ...
# <eof>
```

* 0  line   : configure strings
* 1~ line   : data
* Last line : End of file
* 0  column : points
* 1~ column : data


# Installation

```
$ go get github.com/u1and0/satracli
```


# Licence
MIT

Copyright (c) 2020 u1and0
http://wisdommingle.com/

Permission is hereby granted, free of charge, to any person obtaining a
copy of this software and associated documentation files (the
"Software"), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
