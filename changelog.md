# Changelog

## [Unreleased]
### Added
* Add `font-size-min` to shrink font as necessary to fit
* Add EXIF-based rotation when rotate is not specified
### Changed
* Change `rotate` to default to empty rather than zero
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