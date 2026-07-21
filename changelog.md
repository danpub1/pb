# Changelog

## [Unreleased]
### Added
* Add `font-size-min` to shrink font as necessary to fit
* Add `sort` column setting, sort on exif date, file date, and filename
* Add `{{FileName}}` and `{{FullName}}` in addition to `{{Filename}}` and `{{Fullname}}`
* Add `{{ImageName}}`, `{{ExifDate}}`, and `{{FileDate}}`
* Add FileDate and ExifDate to the `--verbose:X` output
* Add shortcuts `norender`, `nolayout`, `noresize`, `nowatch`, `norecurse`
* Add `day-headers` setting for date sorted files and `{{NextImageDate}}` for its contents
* Add `title` and `subtitle` for creating a title page from the command line
* Add `{{FileDate}}` to display the timestamp used to sort
* Add `deduplicate` book option setting
* Add `text-output-file` text setting to save text to a png or jpeg
* Add sets of images into multiple output files by date
* Add `adjust-by-name` to adjust timestamp of images matching filename
* Add `noprocess`
* Add `{{DATE-HEADER}}` replaceable text for title & subtitle
* Add `max-pages`
* Add `assemble` to assemble other PDFs from output PDFs
### Changed
* Breaking Change: Apply rotation and flip indicated by EXIF orientation
* Detect file date of files in zip files
* Change default size to `size:auto:3x2,3x2`
* Change default page size to 8.5x11"
* Default `size-mode` to the value `area`
* Restructure command line processing to allow no input files when --assemble is present
### Deprecated
### Removed
### Fixed
* Recursing into zip files
* Correctly use row-specific spread-percent when aligning rows
* Margins & Binding
* Don't truncate PDF page sizes to integers


## v1.1.0

### Added
* Support `page-range:*` to select changed pages when using `watch`
* Add `spread-percent`
* Add shortcuts for the values of `distribute-rows`, `distribute-columns`, and `distribute-items`  on page, row, column item types
* Add shortcuts for row-gutter, column-gutter, item-gutter on page, row, column item types
* `page-range:$`: when used with `watch:true`, creates the whole PDF on the first iteration, and then updates the PDF file on subsequent iterations 
* `size:auto`
### Changed
* Default `watch` to true, but do not watch if there is nothing to watch
* Handle multiple books by merging into one
* Help coded into default settings
### Fixed
* Handle errors from `ImageReader.Reader()`
* Check for existence of file before watching a file
* Don't localize paths to embedded files

## v1.0.0

This is the initial public release on github