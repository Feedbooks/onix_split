# onix_split

Command onix_split generates single-product ONIX files from one larger ONIX
file or from a Zip archive containing ONIX files.

`go install githubt.com/feedbooks/onix_split`

## Usage:

`onix_split -file_path path [flag]`

### Flags

`-file_path` *String* ""
  path to the file (_required_)

`-to_files` *Bool* true
  if false, write output to STDOUT

`-pattern` *String*
  process only files that contain this pattern

`-dest_file_prefix` *String* ""
prepended to the generated file names

`-dest_dir` *String* ""
  write output files to this directory
