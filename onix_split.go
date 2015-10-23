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
	"path/filepath"
	"regexp"
	"strings"
)

/* TODO:
- Replace regular expressions with file.ext
- Skip weird Apple files in Zip archives.
*/

func main() {

	var pattern, destDir, destFilePrefix, filePath string

	var total_prods = 0

	var toFiles bool

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
			ext := filepath.Ext(f.Name)
			name_good := ext == ".xml" || ext == ".onix"
			name_bad := f.Name[0] == '.' || f.Name[0] == '_'
			if name_good && !name_bad {
				if pattern == "" {
					buff_reader := ZippedToBuf(f)
					total_prods += split(&buff_reader, toFiles, destDir, destFilePrefix, total_prods)
				} else if match, _ := regexp.MatchString(pattern, f.Name); match {
					buff_reader := ZippedToBuf(f)
					total_prods += split(&buff_reader, toFiles, destDir, destFilePrefix, total_prods)
				}
			}
		}
	} else {
		buff_reader := bufio.NewReader(file)
		total_prods += split(buff_reader, toFiles, destDir, destFilePrefix, total_prods)
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

func split(buff_reader *bufio.Reader, toFiles bool, destDir, destFilePrefix string, total_prods int) int {
	var header_str = ""
	var message_tag = ""
	var product_tag = ""

	leftovers := ""

	buffer_prods := 0

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
		if message_tag == "" {
			message_tag, _ = setTagVariant(&header_str, []string{"ONIXMessage", "ONIXmessage"})
		}
		if product_tag == "" {
			product_tag, _ = setTagVariant(&buff_str, []string{"Product", "product"})
		}

		p_count := strings.Count(buff_str, "</"+product_tag+">")
		arr := strings.Split(buff_str, "</"+product_tag+">")
		leftovers = arr[p_count]
		for i := 0; i < p_count; i++ {
			buffer_prods++
			product_str := fmt.Sprintf("%s\n%s</%s>\n</%s>", header_str, string(arr[i]), product_tag, message_tag)
			if toFiles {
				WriteSplinter(&product_str, destDir, destFilePrefix, total_prods+buffer_prods)
			} else {
				fmt.Println(product_str)
			}
		}
	}
	return buffer_prods
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

func setTagVariant(hay *string, variants []string) (string, error) {
	for _, v := range variants {
		if i := strings.Index(*hay, "<"+v); i > -1 {
			return v, nil
		}
	}
	return "", fmt.Errorf("No %s tag found", variants[0])
}
func setHeader(hay *string) (string, string) {
	header_tag, _ := setTagVariant(hay, []string{"Header", "header"})
	close_header_tag := "</" + header_tag + ">"
	header_end := strings.Index(*hay, close_header_tag) + len(close_header_tag)

	str := string(*hay)
	header_str := str[0:header_end]
	rest_of_hay := str[header_end:]
	return header_str, rest_of_hay
}
