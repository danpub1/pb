# pb

pb is a text-to-photobook conversion tool.
From a marked-up list of photos, it generates a reasonably formatted photobook. Its markup language is intended to support sufficient formatting to make a nicely finished product.

Uses:
* Make a nicely arranged photo book with captions and text describing the photos
* Output contact sheets of all the photos in a folder
* Output a nicely arranged collection of all the photos in a folder - not as small as a contact sheet, but not as finished as a photo book
* Layout pictures on a page in specific sizes for printing
* Create a collage for a calendar page or a greeting card
* Slides for a presentation

## Introduction

The input file format is line-oriented: each line represents one item in the photobook. Some items are visible, like images and blocks of text. Other items are structural: columns, rows, and pages. Items have settings: name-value pairs that determine how they look or work.

A photobook is made up of pages, each of which has rows. Each row has columns, and each column has images and/or text items.

## Content: Images and Text

At its simplest, the input file is a list of images, one per line.

A line with the name of an image file causes the image to be laid out on the current page.

A line with the name of an image file, followed by a `#` (hash or pound sign) and some text is an image with a caption.

A line starting with a `#` with no preceeding image file name is text that will be laid out on the current page.

    Old Faithful.jpeg
    Steamboat Geyser.jpg # Steamboat Geyser
    # Two of the famous geysers in Yellowstone.

## Structure: Book, Pages, Rows, Columns

Content has a default size and is laid out in pages, rows, and columns.

Page-, row-, and column-breaks can be inserted explicitly using these directives on a line by themselves: 

    book: ***
    page: +++
    row: ---
    column: ...

## Additional Directives

    $$$: Defines a style
    @@@: Includes a file

## Settings

Settings are name:value pairs. For a content line, they come after the image (if any) and before the text (if any). 

    Old Faithful.jpeg $ setting:value
    Steamboat Geyser.jpg $ setting:value # Steamboat Geyser
    $ setting:value # Two of the famous geysers in Yellowstone.

For a structure line, they come after the structure token.

    *** setting:value
    +++ setting:value
    --- setting:value
    ... setting:value

## Wildcards

Multiple images can be specified with wildcards (asterisks and question marks).  All images matching the names will be included. 

    Vacation*.jpg

Recursion into subdirectories may also be specified

    *.jpg $ recurse:true

## Blank Lines 

To make the text file easier to read, blank lines are ignored.

## Continuation of Lines 

Any line that begins with one or more spaces is treated as a continuation of the previous line.

## Comments

Any line beginning with three forward slashes should be ignored, so it can be used for notes.

    Old Faithful.jpeg
    Steamboat Geyser.jpg

    # Two of the famous geysers in Yellowstone.
       I saw both of these erupting during my visit.
    /// TODO: go back and figure out what the dates were!

    Old-Faithful.jpeg # The most famous geyser!

## Escaping special characters

The backtick or grave accent is used to escape a few special cases:
* If an image filename starts with a directive (`***`, `+++`, `---`, `...`, `$$$`, `@@@`, `///`), add a `` ` `` before the first character
* A space in an image filename before `$` or `#` must be replaced with `` `_ ``
* A space in a setting value  must be replaced with `` `_ `` (Possible in font name.)
* Backtick must be replaced with two backticks ``` `` ``` in image filenames or settings.
* Internally, `` `_ `` is replaced with space. Anything else after a backtick is replaced with that thing.
* Only image filenames and setting values need to be escaped with backticks, not text after a `#`

Example, first without escaping, then with escaping:
```
---images---\image #.jpg $ font:\folder\Wierd # Name # Old Faithful's #$!@% Erupting
`---images---\image`_#.jpg $ font:\folder\Wierd`_#`_Name # Old Faithful's #$!@% Erupting
```

## Escaping Text

Text contains special characters:
* `\n` is a new line
* `\t` is a tab, used for alignment
* `\1`, `\2`, `\3`, `\4`, `\5`, `\6`, `\7`, `\8`, `\9` select the font in a block of text
* `\0` is a kind of NOP, it does nothing to the final text
* `\\` is a slash

Example:
```
# Hello\nWorld!
# Left\tCenter\tRight
# \1Normal, \2Bold, \3Italic, \4Bold-Italic
```
## Character Set

A pb file is assumed to be UTF-8.

## Styles

Styles are defined using $$$ followed by a space, the name of the style, and the settings being defined:

    $$$ 1 font:Arial.ttf size:9
    $$$ 3 font:Arial.ttf size:11
    $$$ bold font:Arial-Bold.ttf
    $$$ copyright (C) 2026, John Smith. All rights reserved.

Styles are applied by replacing a reference to the style with the defined value of the style.
There are three ways to reference a style:
1. `{{style-name}}` in the settings section or text section of a content line
1. $style-name - equivalent to `$ {{style-name}}`
1. #<number\> or ### (a specific number of hashes)

Styles can reference other styles, as long as they have been defined previously

    $$$ bold3 {{3}} {{bold}}

Examples:

    image.jpg $ {{bold}} size:larger # Caption {{copyright}}
    image.jpg $bold size:larger # Caption {{copyright}}
    image.jpg $bold size:larger #3 Caption {{copyright}}
    image.jpg $bold size:larger ### Caption {{copyright}}

The three sources of settings in the above content lines have the following priority from highest to lowest:
1. The explicit settings in the $ section
1. The settings applied by #number
1. The settings applied by $name

Example:
```
$$$ 1 size:11pt
$$$ 2 font:Arial-Italic.ttf
$$$ 3 font:Arial-Bold.ttf
$$$ 4 font:Arial.ttf

$2 {{3}} # This text is in Arial-Bold.ttf.  The explicit style takes precedence over both the # and $ styles.

$ {{2}} {{1}} {{3}} # This text is in Arial-Bold.ttf.  Equivalent of above.

$2 #4 This text is in Arial.ttf.  The style for # takes precedence over the style for $

$2 # This text is in Arial-Italic.ttf. Style 1 is applied but doesn't specify a font

# This text is in whatever font is specified for the column, row, page, or book.  Style 1 is applied but doesn't specify a font
```
Because styles replace references to themselves, styles that have settings values need to escape those values as needed. Styles used to replace text do not need escaping because text does not need escaping

    $$$ 1 font:Neon`_Sans.ttf

The following lines are equivalent:

    #3 Two of the famous geysers in Yellowstone.
    ### Two of the famous geysers in Yellowstone.

Alignment may also be applied in the #, by appending:

* C for Center
* R for Right
* L for Left
* J for Justified
* B for Binding
* E for Edge

The following lines are roughly equivalent (assuming that what's in style 1 and style 3 are for different settings)

```
$3 align:center # Two of the famous geysers in Yellowstone.

$ {{3}} align:center # Two of the famous geysers in Yellowstone.

$ align:center #3 Two of the famous geysers in Yellowstone.

#3C Two of the famous geysers in Yellowstone.

###C Two of the famous geysers in Yellowstone.
```
Since # is by definition style 1, a typical setup would be to use style 1 for body text, 
style 2 for sub headings, and style 3 for main-headings. Note that this makes the usage of multiple #'s the opposite of Markdown.

Although particularly useful with text, styles can be defined and applied to other directives also, because {{name}} is replaced with everything that was defined for it.

    $$$ book-defaults size:610x820 margin:36
    *** {{book-defaults}}

Styles are just replacement text, and can be used in image filenames as well:

    $$$ dan /home/photos/australia/dan
    $$$ amie /home/photos/australia/amie

    {{dan}}/IMG12345.jpg # Sydney Harbour
    {{amie}}/IMAGE12345.jpg # Koala Bear

### Built-in Styles

There are several pre-defined styles, useful as part of texts in headers and footers:

* `{{Date}}`: Replaced by the current date, not the date of the image file
* `{{Year}}`: Replaced by the four digit year
* `{{Filename}}` Replaced by the name of an image file without the path, useful with wildcards
* `{{Fullname}}`: Replaced by the name of an image file with the path, useful with wildcards
* `{{PageNumber}}`: Replaced by the current page number, useable in headers or footers
* `{{TotalPages}}`: Replaced by the total number of pages in the book, useable in headers or footers

## Special Texts

A text with a setting `name` is not part of the layout, but is used for various purposes, such as header and footer text:

    #1E name:header-first {{title}}\t{{Date}}\t{{PageNumber}}
    #1E name:footer-first Copyright {{Year}}, All rights reserved.

## Tabs

`\t` in a text is a tab.  Two tabs are pre-defined: the first is a center tab, and the second is a right tab.
Prefixing a text with one tab is the equivalent of `align:center`, and prefixing with two tabs is the equivalent of `align:right`.
However, as shown in the example above, tabs are most useful when there is text before the tab.

## Special Images

An image with a setting `name` is not part of the layout, but is used for a page background or image frame.

## Settings

Setting values that are indicated as `yes` or `no` may equivalently be `on` or `off`, `true` or `false`.

Setting values that are colors may be specified as:

* \#A - Equivalent to #AA, i.e. a gray of value AA
* \#BC - Equivalent to #BCBCBCFF, i.e. a gray of value BC
* \#ABC - Equivalent to #AABBCCFF
* \#ABCD - Equivalent to #AABBCCDD, 
* \#89ABCD - Equivalent to #89ABCDFF, an RGB triple with 100% opacity
* \#89ABCDEF - An RGB triple with opacity EF

Settings specified at the book level apply to all the content in the book.  
Settings specified at the page level apply to all the content of the page.  
Settings specified at the row level apply to all the content in the row.  
Settings specified at the column level apply to all the content in the column.  
Settings specified for an individual image or text apply only to that content.  
Some settings only apply at a higher level, and many not be specified lower down.

* `distribute-rows:spreadmiddle`: vertical spacing of rows on the page.  Specifies how extra space is distributed after rows are laid out with gutter in between rows.  One of:
  * `middle`: Divided equally between the top and bottom
  * `justify`: Evenly distributed between rows
  * `top`: Extra is placed at the bottom
  * `bottom`: Extra is placed at the top
  * `binding`: Extra is placed at the binding (if binding is top)
  * `edge`: Extra is placed opposite the binding (if binding is top)
  * `spreadtop`: Distributed between rows and at the bottom
  * `spreadbottom`: Distributed between rows and at the top
  * `spreadmiddle`: Distributed between rows and at the top and bottom
  * `spreadedge`: Distributed between rows and at the binding (if binding is top)
  * `spreadbinding`: Distributed between rows and opposite the binding (if binding is top)

* `size`, `max-size` - size indicates size before resizing, max-size is after resizing
  * Named Relative Sizes - as shortcuts, these may be specified as is without "size:"
    * `normal` - size (default) = scale:1 (this is the same as not specifying size)
    * `larger` - size * 1.25 = scale:1.25
    * `much-larger` - size * 1.25 * 1.25 = size * 1.5625 = scale:1.5625
    * `much-much-larger` - size * 1.25 * 1.25 = size * 1.953125 = scale:1.953125
    * `much-much-much-larger` - size * 1.25 * 1.25 = size * 2.44140625 = scale:2.44140625
    * `smaller` - size / 1.25 = size * 0.8 = scale:0.8
    * `much-smaller` - size / 1.25 / 1.25 = size / 1.5625 = size * 0.64 = scale:0.64
    * `much-much-smaller` - size / 1.25 / 1.25 / 1.25 = size / 1.953125 = size * 0.512 = scale:0.512
    * `much-much-much-smaller` - size / 1.25 / 1.25 / 1.25 / 1.25 = size / 2.44140625 = size * 0.4096 = scale:0.4096
  * `scale:#` - size * #
  * `#` - specific size
  * `#%` - % of total size

{{Settings}}

Image Size
----------

Image sizes are specified relative to a square image, so an image size of 100 specifies an image with an area of 100*100.
If the image being sized has the 3:2 aspect ratio, the area for that width is a = w * w * 2 / 3.
So to get the equivalent 100*100 area, the width needs to be multipled by sqrt(3/2).

Image size is typically set at the page or book level and overridden when desired at the image level by using a relative size (e.g. larger or smaller).

Example: Layout 3:2 images that fit 2 across and 3 down on a 612x792pt page 6 pt gutter between all of them and 36 pt margins
```
Available width = 612 - 36 - 36 - 6 = 534
Want two images in that space = 267
Width needed for a 3:2 image = 267 * sqrt(3 / 2) = 327

Also want 3 image high in that space
Available height = 792 - 36 - 36 - 6 - 6 = 708
Want three images in that space = 236
Height needed for a 3:2 image = 236 * sqrt(2 / 3) = 193
Width for height = 192 * 3 / 2 = 289 

So to fit 3 high it needs to be 289 wide
```

Example: Layout one 3:2 image and one 16:10 image 2 across and 2 down 612x792pt page 6 pt gutter between all of them and 36 pt margins

```
3:2 image: 267 * sqrt(3/2) = 327
16:10 image: 267 * sqrt(16/10) = 338
One of each = harmonic mean
267 * 2 / (1/sqrt(3/2) + 1/sqrt(16/10)) = 332

Similarly 2 down: 357 * sqrt(2/3) = 292, times 3/2 = 437
357 * sqrt(10/16) = 282 * 16/10 = 452

So the harmonic mean of 332 supports one of each across and two rows of either size.
```

## Command Line Options

With the exception of the input file name all command line options may be specified in the input file.  What is provided on the command line
takes precedence.

* `input-file`: Specify the input `.pb` file
* Any setting, applied at the book level (i.e. a book-level default that can be overridden at any lower level)

## TIPS

### Only one book item
There is only one book `***` allowed.  If using an include file, either put the book in the include, or use it to define settings that are applied to the book.

### Points are the default
If using another base measurement, be sure to explicitly define everything in that measurement. Otherwise the default units will be used, which are in points

### Listing Wildcarded Files
To quickly create a list of the files that match a wildcard, use the command line option `-v P`, or the equivalent `>>> v P` in the file.
Redirect the output to another file and cut and paste the listing.

### Clear a setting for a page
To clear the header for a page, for example: `+++ header:`

### Use a text to make a rectangle
```
$ text-width:72 padding:72x0x0x0 font-size:0 text-background:#F0F # 
```

### Use a text to create a blank page
```
+++
#
```

### Aspects not included in layout

Frames, outlines, shadows, and tilts are not included in the layout process

### Align Heading Immediately above body

Use `column-break:false` to put both the header and the body in a column,
and set `item-gutter:0`

### Construct a Cover

Set margins and gutters to zero.  Size one picture to the back cover,
a row of text whose height is the width of the spine, rotates 90 degrees,
and then another picture sized to the front cover.

### Embed an image of a page

Set output-file for a page to a specific name in PNG format,
then reference that image in a later page

### Example Command Line:

```
pb Selected.zip --page-break:true --font:Aptos.zip::Aptos.ttf --caption:{{Filename}} --page-size:612x792 --output-file:big.pdf --verbose:D
```

## Things to Do

* Issue: Had problems when title pages were first, like first page cannot have some or all settings. Also first row, column, item???
* Refactor & clean up
  * Break up large files
  * Latest dependencies
  * Tests
  * Any other go-novice mistakes
* Support input from files better so everything can be done by dropping files on the app
    * Redirect stdout with a book-level setting - makes it possible to create both pdf and .pb files in one command without options
* size:auto to make the necessary calculations
* Output a text as an image (can do already, need to get height from verbose output)
* Input and output handlers for more file types
* Sigmoidal brightness/lightness Adjustment? (i.e. sigmoidal but on a different channel)
* Highlights, midtones, shadows Adjustment
* HSL Adjustment
* Image, font https://... downloaded and then cached (in a zip file?)
* Calendar pages
* UI of its own - ebitengine or web browser-based?  Launch pdf or image viewer?
* Colorspace
* HDR