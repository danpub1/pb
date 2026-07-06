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
* Add `{{FileDate}}` f
### Changed
* Breaking Change: Apply rotation and flip indicated by EXIF orientation
* Detect file date of files in zip files
* Change default size to `size:auto:3x2,3x2`
* Change default page size to 8.5x11"
### Deprecated
### Removed
### Fixed

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