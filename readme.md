# pb

pb is a text-to-photobook conversion tool.
From a marked-up list of photos, it generates a reasonably formatted photobook. Its markup language is intended to support sufficient formatting to make a nicely finished product.

Uses:
* Make a nicely arranged photo book with captions and text describing the photos
* Make contact sheets of all the photos in a folder
* Make a nicely arranged collection of all the photos in a folder - not as small as a contact sheet, but not as finished as a photo book
* Layout pictures on a page in specific sizes for printing
* Create a collage for a calendar page or a greeting card
* Create slides for a presentation

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

    ***: Book Settings
    +++: Page Break & Page Settings
    ---: Row Break & Row Settings
    ...: Column Break & Column Settings

Additional directives:

    $$$: Style Definition
    @@@: Include another file

## Settings

Settings are name:value pairs. For a content line, they come after the image (if any) and before the text (if any), preceded by a `$`.

    Old Faithful.jpeg $ size:larger
    Steamboat Geyser.jpg $ size:larger font-size:12 # Steamboat Geyser
    $ font-size:12 # Two of the famous geysers in Yellowstone.

For a structure line, they come after the structure token.

    *** units:pt
    +++ margin:24
    --- column-gutter:6
    ... distribute-items:spreadmiddle

## Wildcards

Multiple images can be specified with wildcards (asterisks and question marks).  All images matching the names will be included. 

    Vacation*.jpg

Recursion into subdirectories may also be specified

    *.jpg $ recurse:true

## Blank Lines 

To make the text file easier to read, blank lines are ignored.

## Continuation of Lines 

Any line that begins with one or more spaces is treated as a continuation of the previous line.
The previous line's trailing spaces and the continuation line's leading spaces are collapsed. 

## Comments

Any line beginning with three forward slashes should be ignored, so it can be used for notes.

    Old Faithful.jpeg
    Steamboat Geyser.jpg

    # Two of the famous geysers in Yellowstone.
       I saw both of these erupting during my visit.
    /// TODO: go back and figure out what the dates were!

    Old-Faithful.jpeg # The most famous geyser!

## Escaping special characters

The backtick or grave accent, `` ` ``, is used to escape a few special cases:
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

Text after a `#` may contain special characters:
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

Styles are defined using `$$$` followed by a space, the name of the style, and the settings being defined:

    $$$ 1 font:Arial.ttf size:9
    $$$ 3 font:Arial.ttf size:11
    $$$ bold font:Arial-Bold.ttf
    $$$ copyright (C) 2026, John Smith. All rights reserved.

Styles are applied by replacing a reference to the style with the defined value of the style.
There are three ways to reference a style:
1. `{{style-name}}` in the settings section or text section of a content line
1. `$style-name` - equivalent to `$ {{style-name}}`
1. `#number` or `###` (a specific number of hashes)

Styles can reference other styles, as long as they have been defined previously

    $$$ bold3 {{3}} {{bold}}

Examples:

    image.jpg $ {{bold}} size:larger # Caption {{copyright}}
    image.jpg $bold size:larger # Caption {{copyright}}
    image.jpg $bold size:larger #3 Caption {{copyright}}
    image.jpg $bold size:larger ### Caption {{copyright}}

The three sources of settings in the above content lines have the following priority from highest to lowest:
1. The explicit settings in the `$` section
1. The settings applied by `#number`
1. The settings applied by `$name`

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

Alignment may also be applied in the `#`, by appending:

* `C` for Center
* `R` for Right
* `L` for Left
* `J` for Justified
* `B` for Binding
* `E` for Edge

The following lines are roughly equivalent (assuming that what's in style 1 and style 3 are for different settings)

```
$3 text-align:center # Two of the famous geysers in Yellowstone.

$ {{3}} text-align:center # Two of the famous geysers in Yellowstone.

$ text-align:center #3 Two of the famous geysers in Yellowstone.

#3C Two of the famous geysers in Yellowstone.

###C Two of the famous geysers in Yellowstone.
```
Since `#` is by definition style 1, a typical setup would be to use style 1 for body text, 
style 2 for sub headings, and style 3 for main-headings. Note that this makes the usage of multiple `#`'s the opposite of Markdown.

Although particularly useful with text, styles can be defined and applied to other directives also, because `{{name}}` is replaced with everything that was defined for it.

    $$$ book-defaults size:610x820 margin:36
    *** {{book-defaults}}

Styles are just replacement text, and can be used in image filenames as well:

    $$$ phone /home/photos/australia/phone
    $$$ camera /home/photos/australia/camera

    {{phone}}/IMG12345.jpg # Sydney Harbour
    {{camera}}/IMAGE12345.jpg # Koala Bear

### Built-in Styles

There are several pre-defined styles, useful as part of texts in headers and footers:

* `{{Date}}`: Replaced by the current date, not the date of the image file
* `{{Year}}`: Replaced by the four digit year
* `{{Filename}}` or `{{FileName}}` Replaced by the name of an image file without the path, useful with wildcards
* `{{Fullname}}` or `{{FullName}}`: Replaced by the name of an image file with the path, useful with wildcards
* `{{ImageName}}`: Replaced by the full name of the image as given in the file
* `{{ExifDate}}`: Replaced by the date as found in the EXIF metadata
* `{{FileDate}}`: Replaced by the timestamp of the file
* `{{ImageDate}}`: Replaced by the best-guess timestamp of the image - based on the filename, EXIF date, and modified time of the file
* `{{PageNumber}}`: Replaced by the current page number, useable in headers or footers
* `{{TotalPages}}`: Replaced by the total number of pages in the book, useable in headers or footers
* `{{NextImageDate}}`: Replaced by the date only of the next image following the text.  For use with `day-headers`

## Special Texts

A text with a setting `name` is not part of the layout, but is used for various purposes, such as header and footer text:

    #1E name:header-first {{title}}\t{{Date}}\t{{PageNumber}}
    #1E name:footer-first Copyright {{Year}}, All rights reserved.

## Tabs

`\t` in a text is a tab.  Two tabs are pre-defined: the first is a center tab, and the second is a right tab.
Prefixing a text with one tab is the equivalent of `align:center`, and prefixing with two tabs is the equivalent of `align:right`.
However, as shown in the example above, tabs are most useful when there is text before the tab.

## Named Items

An image or text with a setting `name` is not part of the layout, but is used for a page background or image frame, or for header or footer text.

## Settings

Settings specified at the book level apply to all the content in the book.  
Settings specified at the page level apply to all the content of the page.  
Settings specified at the row level apply to all the content in the row.  
Settings specified at the column level apply to all the content in the column.  
Settings specified for an individual image or text apply only to that content.  
Some settings only apply at a higher level, and many not be specified lower down.

***NOTE:*** Settings specified at the page, row, or column level apply to subsequent automatically created pages, rows, or columns until explicitly ended with another page, row, or column directive.

Setting values that are indicated as `yes` or `no` may equivalently be `on` or `off`, or `true` or `false`.

Setting values that are colors may be specified as:

* `#A` - Equivalent to `#AA`, i.e. a gray of value 0xAA
* `#BC` - Equivalent to `#BCBCBCFF`, i.e. a gray of value 0xBC
* `#ABC` - Equivalent to `#AABBCCFF`
* `#ABCD` - Equivalent to `#AABBCCDD`, 
* `#89ABCD` - Equivalent to `#89ABCDFF`, an RGB triple with 100% opacity
* `#89ABCDEF` - An RGB triple with opacity 0xEF

### Arranging rows, columns, and items

There are three `distribute-` settings which specify how to use the blank space left over after resizing images to fit the page.

`distribute-rows` specifies how extra space is distributed between rows and at the top and bottom of the page after rows are laid out with gutter in between rows.

`distribute-columns` specifies how extra space is distributed between columns and the the left and right side of the page after the columns are laid out with gutter between them.

`distribute-items` specifies how extra space is distributed between items in a column.

The three settings take the following values:

  * `middle`: Divided equally between the top and bottom
  * `center`: Divided equally between the left and right
  * `justify`: Evenly distributed between with none at the edges
  * `top`: Extra is placed at the bottom
  * `bottom`: Extra is placed at the top
  * `left`: Extra is placed at the right
  * `right`: Extra is placed at the left
  * `binding`: Extra is placed at the binding
  * `edge`: Extra is placed opposite the binding
  * `spreadtop`: Distributed between and at the bottom
  * `spreadbottom`: Distributed between and at the top
  * `spreadleft`: Distributed between and at the right
  * `spreadright`: Distributed between and at the left
  * `spreadmiddle`: Distributed between and at the top and bottom
  * `spreadcenter`: Distributed between and at the left and right
  * `spreadedge`: Distributed between and at the binding
  * `spreadbinding`: Distributed between and opposite the binding

### Image Size

* `size` indicates image size before resizing
* `max-size` indicates image size after resizing
* There are different ways of specifying size:
  * An explicit number of units, `size:250`
  * A percentage of the page size, `size:25%`
  * A size relative to the default size
    * `size:scale:#` - the default size times a scale factor
    * `size:normal` - equivalent to `size:scale:1`
    * `size:larger` - equivalent to `size:scale:1.25`
    * `much-larger` - equivalent to `size:scale:1.5625` (1.25 * 1.25)
    * `much-much-larger` - equivalent to `size:scale:1.953125` (1.25 * 1.25 * 1.25)
    * `much-much-much-larger` - equivalent to `size:scale:2.44140625` (1.25 * 1.25 * 1.25 * 1.25)
    * `smaller` - equivalent to `size:scale:0.8` (divide by 1.25)
    * `much-smaller` - equivalent to `size:scale:0.64` (divide by 1.25 * 1.25)
    * `much-much-smaller` - equivalent to `size:scale:0.512` (divide by 1.25 * 1.25 * 1.25)
    * `much-much-much-smaller` - equivalent to `size:scale:0.4096` (divide by 1.25 * 1.25 * 1.25 * 1.25)
  * A size calculated to fit items across a page, `size:auto:3:2,2:3,3:2`

There are two modes of specifying size:

* `size-mode:width` - the size is treated as the width of the image
* `size-mode:area` - the size is treated as the width of a square image, and the image being sized given the same area.


Book-Level Settings
-------------------
* `output-gamma`: Apply a gamma correction to the page bitmap. This is useful to lighten or darken printed output so it better matches the onscreen experience. [1.0]
* `output-sharpen`: Apply sharpening to the page bitmap after resizing is complete. [0.0]
* `day-headers`: Either `auto` or a named text to use as day headers. []
* `density`: Pixels per unit when converting the content to a page bitmap.  2 pixels per pt (144 ppi) could be considered for a preview quality, and 5 pixels per pt (360 ppi) could be appropriate for printing. [2.0]
* `binding`: The book's binding location, one of `side`, `top`, `none`.  Controls if margins are alternated by even/odd pages. [side]
* `output-compression`: The jpeg compression level when creating the page bitmap. [92]
* `output-mozjpeg-sampling`: The subsampling used by mozjpeg. Typically one of: `1x1` (4:4:4), `1x2` (4:4:0), `2x1` (4:2:2), `2x2` (4:2:0), `4x1` (4:1:1), `4x2` (4:1:0). [1x1]
* `units`: The units of measure used in laying out the book.  One of `in`, `cm`, `mm`, `pt`. [pt]
* `output-mozjpeg`: Use the mozjpeg compressor to create the page bitmap. Slower, but produces smaller files at the same quality. [false]

Page-Level Settings
-------------------
* `background`: Color or named image to use as the page background. [#F]
* `row-gutter`: Gutter, in units, between rows. [6.0]
* `current-page`: Include this page in the output regardless of whether it included in the page range. [false]
* `header`: Name of even page text, Name of odd page text, Offset from the margin in units, Number of Leading Pages without Page Numbers, Number of Trailing Pages without Page Numbers. Example: `EvenHeader,OddHeader,2.5,2,2`. The header names are named text items which are offset above or below the top margin by the offset. Page numbers and total pages are calculated assuming some unnumbered pages before and after. []
* `margin`: Page margin in units, All, Top & Bottom x Left & Right, Top x Right x Bottom x Left. Examples: `24`, `24.5x18.1`, `36x24x36x12`. When binding is `side`, the right margin starts out as binding and the left margin is the edge.  When binding is `top`, the top margin is initially the binding and the bottom margin is the edge. [24.0]
* `page-size`: Page size in units, width x height. [612.0x792.0]
* `footer`: Same as `header`, but with text offset below the bottom margin. []
* `distribute-rows`: Vertical spacing of rows on the page.  Specifies how extra space is distributed. [spreadmiddle]

Row-Level Settings
------------------
* `column-gutter`: Gutter, in units, between columns. [6.0]
* `distribute-columns`: Horizontal spacing of columns in the row.  Specifies how extra space is distributed. [spreadcenter]

Column-Level Settings
---------------------
* `item-gutter`: Gutter, in units, between items in a column. [6.0]
* `keep-columns-together`: If a column becomes too wide and there is not room for it in the next row, move it as a unit to the next page, versus breaking the page and starting a new page with the item. [false]
* `distribute-items`: Vertical spacing of items in a column.  Specifies how extra space is distributed. [spreadmiddle]

Column/Row/Page-Level Settings
------------------------------
* `spread-percent`: How `spreadmiddle` or `spreadcenter` spreads extra at top/bottom or left/right. Useful for positioning one item 1/3 of the way down the page instead of at the center. [50.0]

Column/Row/Page/Book-Level Settings
-----------------------------------
* `sort`: Sort consecutive images. `date` sorts by date first and then by filename, `filename` sorts only by filename, `none` or the empty string retains the given order. [none]

Image/Text-Level Settings
-------------------------
* `column-break`: Break the column before this item. [true]
* `page-break`: Break the page before this item. [false]
* `row-break`: Break the row before this item. [false]
* `name`: Define a named text or image, to be used as a background, header, or footer. []
* `float`: For text, horizontal and vertical offset in units from the left and top margins. For images, also the image width and height. (Text width and height are a function of the text and the font size.). []
* `corner-radius`: Give the item a superellipse-based corner. Size in units, optional power.  If power is less than zero, the corner bends inward.  If power is equal to zero, the corner is straight.  If power is one, the corner is circular.  If power is greater than one, the corner is flatter than circular. Example: `100%,2`, `2.5,1.725`. [0.0]
* `item-align`: How to align this item in a column versus other items in the colum.  One of: `left`, `center`, `right`. `binding`, `edge`. [center]
* `tilt`: Rotate the item this many degrees. Use this for smaller rotations, and `rotate` for bigger rotations. [0.0]
* `rotate`: Rotate this item around its center.  One of `0`, `90`, `180`, `270`. Added to whatever rotation is indicated by EXIF orientation, if any. [0]

Image-Level Settings
--------------------
* `brightness`: Adjust image brightness, -100.0 to 100.0. [0.0]
* `sigmoidal`: Adjust image contrast with sigmoidal function, specifying factor and midpoint. [0.0,0.50]
* `contrast`: Adjust image contrast, -100.0 to 100.0. [0.0]
* `gamma`: Adjust image gamma. [1.0]
* `saturation`: Adjust image saturation, -100.0 to 100. [0.0]
* `sharpen`: Adjust image's sharpness. [0.0]
* `blur`: Blur the image. [0.0]
* `caption`: Caption Text. Typically not specified using this setting. []
* `flip`: Flip the image either horizontally (H) or vertically (V). []
* `size-mode`: How to interpret size: as `width` or as width relative to `area` of square. [area]
* `image-background`: Image background color. [#0000]
* `image`: Image filename. Typically not specified using this setting. []
* `image-frame`: Image frame size in units (all, top+bottomxleft+right, topxrightxbottomxleft), color or image name, position (above or below). [0]
* `image-outline`: Image outline color, size in units. []
* `rect`: Image rectangle: fit,aspect-ratio,percent; trim,aspect-ratio,percent; squish,aspect-ratio; zoom-level,horizontal offset,vertical offset,aspect-ratio,percent. Aspect-ratio is image aspect ratio if not specified. Percent is 50 if not specified. [100]
* `image-shadow`: Image shadow color, blur, and horizontal and vertical offset in units. []
* `size`: Initial image size (before resizing). [auto:3x2,3x2]
* `max-size`: Maximum image size (after resizing). [100%]
* `recurse`: Recurse directories when matching wildcard images. [true]
* `straighten`: Straighten the image by specified angle, maximizing the image's rectangle. [0.0]

Text-Level Settings
-------------------
* `breakchars`: Additional characters to allow line-breaks. []
* `caption-align`: Alignment of the caption: left, center, right, justified. [center]
* `caption-position`: Caption position above or below. [below]
* `caption-squareness`: Defines whether the caption's width is based on the image width (100) or the larger dimension (0). [100]
* `font`: Font name. []
* `font-size`: Font size, units. [11.0]
* `caption-gutter`: Gutter, in units, between image and caption. [0.0]
* `text-height`: Height of text block, units. [0]
* `font-size-min`: If greater than zero, reduce font-size to this as needed to fit. [0]
* `letterspacing`: Letter spacing, units. [0.0]
* `linespacing`: Line spacing, multiple of lines. [1.0]
* `padding`: Padding around text block, (all, top+bottom,left-right, top+right+bottom+left). [3.5]
* `text-align`: Text alignmnet: left, center, right, justified. [left]
* `text-background`: Text background color. [#0000]
* `text-color`: Text color. [#0]
* `text-frame`: Text frame size in units (all, top+bottomxleft+right, topxrightxbottomxleft), color. [0.0]
* `text-outline`: Text outline color, size in units. []
* `text-shadow`: Text shadow color, blur, and horizontal and vertical offset in units. []
* `text-width`: Text width. [100%]
* `text-wrap`: Text wrapping mode: balanced/unbalanced. [unbalanced]
* `text`: Text. Typically not specified using this setting. []
* `justify-weight`: Weight to give spaces versus letters when justifying text. [2.5]
* `wordspacing`: Word spacing, units. [0.0]

Book-Level Options
------------------
* `cache-mode`: Controls Image Cache. 0=Do not cache, 1=Cache during a run but flush cache at beginning of run, 2=Fully cache image measurements across runs. [0]
* `page-range`: Include the specified pages in the output. `*` means only changed pages, `$` means changed pages but update PDF.  Examples: `1-10,50-`, `1-2,*`. [$]
* `watch`: Regenerate the output when the input file changes, versus generate the output once and then exit. [true]
* `verbose`: Zero or more of D, P, X, L.  D=Details, P=Print, X=Print with comments, L=Verbose Logging. [D]

Page-Level Options
------------------
* `output-file`: . [out.pdf]
* `nolayout`: For debugging. After distributing images to pages, do not distribute them in the freee space. [false]
* `noresize`: For debugging. After distributing images to pages, do not resize them to fill the page. [false]
* `norender`: For debugging. Do not render this page to a bitmap. [false]
* `cjpeg-command`: Path to cjpeg executable, e.g. `/usr/bin/cjpeg`. []


### Settings Shortcuts

The following settings have shortcuts:

* `rect` Settings
  * `rect:trim:settings` => `trim:settings`
  * `rect:fit:settings` => `fit:settings`
  * `rect:squish:settings` => `squish:settings`
  * `rect:settings` => `crop:settings`
* `size` Settings
  * `size:scale:value` => `scale:value`
  * `size:larger` => `larger`
  * `size:much-larger` => `much-larger`
  * `size:much-much-larger` => `much-much-larger`
  * `size:much-much-much-larger` => `much-much-much-larger`
  * `size:smaller` => `smaller`
  * `size:much-smaller` => `much-smaller`
  * `size:much-much-smaller` => `much-much-smaller`
  * `size:much-much-much-smaller` => `much-much-much-smaller`
* `gutter` Settings
  * On a page, `row-gutter:value` => `gutter:value`
  * On a row, `column-gutter:value` => `gutter:value`
  * On a column, `item-gutter:value` => `gutter:value`
* Default-true Settings
  * `page-break:true` => `page-break`
  * `row-break:true` => `row-break`
  * `column-break:true` => `column-break`
  * `current-page:true` => `current-page`
  * `noresize:true` => `noresize`
  * `nolayout:true` => `nolayout`
  * `norender:true` => `norender`
  * `watch:false` => `nowatch`
  * `recurse:false` => `norecurse`
* `distribute-` Settings
  * On a page, `distribute-rows:value` => value
  * On a row, `distribute-columns:value` => value
  * On a column, `distribute-items:value` => value
  * Values
    * `spreadtop` / `spreadmiddle` / `spreadbottom`
    * `spreadleft` / `spreadcenter` / `spreadright`
    * `spreadbinding` / `spreadedge`
    * `top` / `middle` / `bottom`
    * `left` / `center` / `right`
    * `binding` / `edge`
    * `justify`
* `verbose:H` => `help`

## Command Line Options

* `input-file`: Specify the input `.pb` file.  Multiple files may be specified and are processed in the order listed.  Zip files may be specified and are treated as a container of images.
* Any setting may be applied at the book level by prepending it with two hypens.  For example: `--page-size:576x576`

## TIPS

### Only one book item
There is really only one book `***`.  If there are multiple books specified, they are merged into one in the order they are found.

### Points are the default
If using another base measurement, be sure to explicitly define everything in that measurement. Otherwise the default units will be used, which are in points

### Listing Wildcarded Files
To quickly create a list of the files that match a wildcard, use the command line option `--verbose:P`, or the equivalent `verbose:P` in the file. Redirect the output to another file and cut and paste the listing.

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

Use `column-break:false` to put both a header text and a body text in a column,
and set `item-gutter:0`

### Construct a Cover

Set margins and gutters to zero.  Size one picture to the back cover,
a row of text whose height is the width of the spine, rotated 90 degrees,
and then another picture sized to the front cover.

### Embed an image of a page

Set output-file for a page to a specific name in PNG format,
then reference that image in a later page

### Example Command Line:

```
pb Selected.zip --page-break:true --font:Aptos.zip::Aptos.ttf --caption:{{Filename}} --page-size:612x792 --output-file:big.pdf --verbose:D
```

### Full Collection by Date

```
pb Collection1.zip Collection2.zip --caption:{{ImageName}} --nowatch --sort:date --day-headers:auto --max-size:75% --size-mode:area --distribute-rows:spreadtop
```

## Things to Do

* Issue: Had problems when title pages were first, like first page cannot have some or all settings. Also first row, column, item???
* Issue: text-background does not work with text-outline
* Issue: Column overflow creates endless loop
* Refactor & clean up
  * Break up large files
  * Latest dependencies
  * Tests
  * Any other go-novice mistakes
* Support input from files better so everything can be done by dropping files on the app
    * Redirect stdout with a book-level setting - to create both pdf and .pb files in one command without options
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
