/* Command onix_split generates single-product ONIX files from one larger ONIX
 file or from a Zip archive containing ONIX files.

 	Usage:

 		onix_split -file_path path [flag]

	The flags are:

		-file_path String, ""
			path to the file (required)
		-to_files Bool, true
			if false, write output to STDOUT
		-pattern String
			process only files that contain this pattern
		-dest_file_prefix String, ""
			prepended to the generated file names
		-dest_dir String String, ""
			write output files to this directory
*/
package main

import (
	"archive/zip"
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

/* TODO:
- Validate first generated product xml.
- Option to validate each generated product xml.
*/

var pattern, destDir, destFilePrefix, filePath string
var toFiles bool

var total_prods = 0

func main() {

	flag.StringVar(&filePath, "file_path", "", "Path to the file to split")
	flag.StringVar(&pattern, "pattern", "", "A pattern to match when file_path is to a multi-file archive")
	flag.StringVar(&destDir, "dest_dir", "./", "Destination directory")
	flag.StringVar(&destFilePrefix, "dest_file_prefix", "", "Prepended to the generated file names")

	flag.BoolVar(&toFiles, "to_files", true, "Write output to files")

	flag.Parse()
	if filePath == "" {
		panic("You must provide a file path")
	}

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}()

	if filePath[len(filePath)-4:] == ".zip" {
		r, err := zip.OpenReader(filePath)
		if err != nil && err != io.EOF {
			panic(err)
		}
		defer r.Close()
		for _, f := range r.File {
			if ix_xml, _ := regexp.MatchString(".xml$", f.Name); ix_xml {
				if pattern == "" {
					buff_reader := ZippedToBuf(f)
					Split(&buff_reader)
				} else if match, _ := regexp.MatchString(pattern, f.Name); match {
					buff_reader := ZippedToBuf(f)
					Split(&buff_reader)
				}
			}
		}
	} else {
		buff_reader := bufio.NewReader(file)
		Split(buff_reader)
	}

	fmt.Printf("Collected %d products\n", total_prods)
}

func ZippedToBuf(f *zip.File) bufio.Reader {
	unzipped, err := f.Open()
	if err != nil {
		panic(err)
	}
	buff_reader := bufio.NewReader(unzipped)
	return *buff_reader
}

func Split(buff_reader *bufio.Reader) {
	var header_str = ""
	var product_tag = ""

	leftovers := ""

	for {
		buff := make([]byte, 2<<20)
		n, err := buff_reader.Read(buff)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if n == 0 {
			break
		}

		buff_str := leftovers + string(buff[:n])

		if header_str == "" {
			header_str, buff_str = setHeader(&buff_str)
		}
		if product_tag == "" {
			product_tag = SetTag("product", &buff_str)
		}

		p_count := strings.Count(buff_str, "</"+product_tag+">")
		arr := strings.Split(buff_str, "</"+product_tag+">")
		leftovers = arr[p_count]
		for i := 0; i < p_count; i++ {
			total_prods++
			product_str := fmt.Sprintf("%s\n%s</%s>\n</ONIXmessage>", header_str, string(arr[i]), product_tag)
			if toFiles {
				WriteSplinter(&product_str, destDir, destFilePrefix, total_prods)
			} else {
				fmt.Println(product_str)
			}
		}
	}
}

func WriteSplinter(onix *string, directory, prefix string, serial int) {
	full_path := fmt.Sprintf("%s/%s%d.xml", directory, prefix, serial)
	splinter, err := os.Create(full_path)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := splinter.Close(); err != nil {
			panic(err)
		}
	}()
	splinter.WriteString(*onix)
}

// SetTag determines which format (lower case or title) of a tag is used the file.
// It returns the tag in correct format.
func SetTag(tag string, hay *string) string {
	lc := strings.ToLower(tag)
	title := strings.ToTitle(tag)
	found := -1

	if found = strings.Index(*hay, "<"+lc+">"); found > 0 {
		return lc
	} else if found = strings.Index(*hay, "<"+title+">"); found > 0 {
		return title
	} else {
		panic("No " + tag + " tag found")
	}
}

func setHeader(hay *string) (string, string) {
	close_header_tag := "</" + SetTag("header", hay) + ">"
	header_end := strings.Index(*hay, close_header_tag) + len(close_header_tag)

	str := string(*hay)
	header_str := str[0:header_end]
	rest_of_hay := str[header_end+1 : len(str)]
	return header_str, rest_of_hay
}
