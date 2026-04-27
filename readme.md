# pb

pb is a text-to-photobook conversion tool.
From a marked-up list of photos, it generates a reasonably formatted photobook. Its markup language supports sufficient formatting to make a nicely finished product.

Uses:
* Make a nicely arranged photo book with captions and text describing the photos
* Output a contact sheet of all the photos in a folder
* Output a nicely arranged collection of all the photos in a folder - not as small as a contact sheet, but not as finished as a photo book
* Layout pictures on a page in specific sizes for printing
* Create a collage for a calendar page or a greeting card

## Introduction

The input file format is line-oriented: each line represents one item in the photobook. Some items are visible, like images and blocks of text. Other items are structural: columns, rows, and pages. Items have settings: name-value pairs that determine how they look or work.

A photobook is made up of pages, each of which has rows. Each row has columns, and each column has images and/or text item.

### Content: Images and Text

A line with the name of an image file causes the image to be laid out on the current page.

A line starting with a `#` (hash or pound sign) is text that will be laid out on the current page.

    Old Faithful.jpeg
    Steamboat Geyser.jpg
    # Two of the famous geysers in Yellowstone.

### Captions

Normally, a block of text goes all the way across the page. To make a text that corresponds to an image, include it on the same line:

    Steamboat Geyser.jpg # Steamboat Geyser is one of the famous ones in Yellowstone.
    Old Faithful's Erupting.jpeg # Old Faithful's Erupting!

### Settings

For an image with a caption, settings come between the image and the caption:

    Old Faithful's Erupting.jpeg $ straighten:1.0 font:Times.ttf # Old Faithful's Erupting!

For an image without a caption, settings come after the image:

    Old Faithful's Erupting.jpeg $ straighten:1.0

For a text without an image, settings come before:

    $ font:Times.ttf # Old Faithful's Erupting!

### Wildcards

Multiple images can be specified with wildcards (asterisks and question marks).  All images matching the names will be included. 

    Vacation*.jpg

Recursion into subdirectories may also be specified

    *.jpg $ recurse:true # {{Filename}}

### Blank Lines 

To make the text file easier to read, blank lines are ignored.

### Continuation of Lines 

Any line that begins with one or more spaces is treated as a continuation of the previous line.

### Comments

Any line beginning with three forward slashes should be ignored, so it can be used for notes.

    Old Faithful.jpeg
    Steamboat Geyser.jpg

    # Two of the famous geysers in Yellowstone.
       I saw both of these erupting during my visit.
    /// TODO: go back and figure out what the dates were!

    Old-Faithful.jpeg # The most famous geyser!

### Escaping special characters

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

### Escaping Text

Text contains special characters:
* `\n` is a new line
* `\t` is a tab, used for alignment
* `\1`, `\2`, `\3`, `\4` select the font in a block of text
* `\0` is a kind of NOP, it does nothing to the final text
* `\\` is a slash

Example:
```
# Hello\nWorld!
# Left\tCenter\tRight
# \1Normal, \2Bold, \3Italic, \4Bold-Italic
```
### Character Set

A pb file is assumed to be UTF-8.

## Structural Items and Directives: Formatting & Styling Content

`***` = A book and its settings.  There must only be one in the file  
`+++` = A page and its setetings  
`---` = A row and its settings  
`...` = A column and its settings  
`$$$` = defines style  
`@@@` = includes another pb file  
`///` = comment  
`>>>` = an option

Given a list of images and text, pb arranges them into columns and rows and pages. ***How*** they are arranged is controlled through settings.

    *** size:621x810 margin:36

    Old Faithful's Erupting.jpeg
    Steamboat Geyser.jpg
    # Two of the famous geysers in Yellowstone.

The structural items (book, page, row, column) have settings (in the example above, size and margin are the settings), and settings have names and values 
(the value of the size setting above is 621x810).  Settings have default values, which are used if the setting is not
specified.  For example, the setting 'units' has a default value of 'pt' (points).  Since it was not specified in the book directive,
the size is assumed to be in points, 621 points wide by 810 points tall (or 8.625 x 11.25 inches).

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

# Settings

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

Book-Level Settings
-------------------

* `units:pt`: the units of measure used in laying out the book.  One of `in`, `cm`, `mm`, `pt`
* `density:2.0`: pixels per unit when converting the content to a page bitmap.  2 pixels per pt (144 ppi) could be considered for a preview quality, and 5 pixels per pt (360 ppi) could be appropriate for printing.
* `binding:side`: the book's binding location, one of `side`, `top`, `none`.  Controls if margins are alternated by even/odd pages
* `output-gamma:1.0`: apply a gamma correction to the page bitmap. This is useful to lighten or darken printed output so it better matches the onscreen experience.
* `output-sharpen:0.0`: apply sharpening to the page bitmap after resizing is complete.
* `output-compression:92`: the jpeg compression level when creating the page bitmap
* `output-mozjpeg:false`: use the mozjpeg compressor to create the page bitmap.
* `output-mozjpeg-sampling:1x1`: the subsampling used by mozjpeg

Page-Level Settings
-------------------

* `size:576x576`: Page size, width x height, in units.
* `margin:0.0x0.0x0.0x0.0`: Margins in the order Top, Right, Bottom, Left (binding:none), Top, Binding, Bottom, Edge (for binding:side), or Binding Right Edge Left (binding:top).  In units or percent.
* `margin:0.0x0.0`: Margins in the order Top/Bottom, Right/Left
* `margin:0.0`: Margins, all the same
* `background:#color/name`: Color or named image to use as the page background
* `header:even-header,odd-header,offse-from-margin,t,leading-pages,trailing-pages`: Named text to use as the header
* `footer:even-footer,odd-footer,offset-from-margin,,leading-pages,trailing-pages`:  Named text to use as the header
* `current-page`: false.  Set to true to output the page when previewing. As a shortcut, `current-page` by itself is equivalent to `current-page:true`
* `row-gutter:6.0`:
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

Row-Level Settings
------------------
* `column-gutter:6.0`:
* `distribute-columns:spreadcenter`: horizontal spacing of columns in a row.  Specifies how extra space is distributed after columns are laid out with gutter in between columns.  One of:
  * `center`: Divided equally between the left and right
  * `justify`: Evenly distributed between columns
  * `left`: Extra is placed at the right
  * `right`: Extra is placed at the left
  * `binding`: Extra is placed opposite the binding (if binding is side)
  * `edge`: Extra is placed at the edge (if binding is side)
  * `spreadleft`: Distributed between columns and at the right
  * `spreadright`: Distributed between columns and at the left
  * `spreadmiddle`: Distributed between columns and at the left and right
  * `spreadedge`: Distributed between columns and at the binding (if binding is side)
  * `spreadbinding`: Distributed between rows and opposite the binding (if binding is side)

Column-Level Settings
---------------------
* `distribute-item`: vertical spacing of items in a column (same values as for page-distribute)
* `item-gutter:6.0`: 
* `keep-columns-together:false`: indicates how to break columns when they overflow row width. 
If a column is too wide, should it be broken and continued on the next page, 
or should the whole column be moved to the next page?

Image Settings
--------------
* `straighten:0.0`: Rotates an image within its frame
* `brightness:0.0`: Applies brightness adjustment, -100 to 100
* `contrast:0.0`: Applies contrast adjustment, -100 to 100
* `gamma:1.0`: Applies gamma adjustment
* `saturation:0.0`: Applies saturation adjustment, -100 to 100
* `sigmoidal:0.0,0.5`: Applies sigmoidal contrast adjustment. -10.0 < factor < 10.0, 0.0 <= midpoint <= 1.0
* `trim:w:h,50`: (shortcut for rect:trim:w:h,50)
* `fit:w:h,50`: (shortcut for rect:fit:w:h,50)
* `squish:w:h` (shortcut for rect:squish:w:h)
* `crop:100,0,0,#:#,50` (shortcut for rect:100,0,0,#:#,50)
* `rect:trim:#:#,50`: Trim & Crop to fit in aspect ratio (Center, Left/Top, Right/Bottom, 0-100% [50%])
* `rect:fit:#:#,50`: Shrink to fit in aspect ratio (Center, Left/Top, Right/Bottom, 0-100% [50%]))
* `rect:squish:#:#`: Fit to specified aspect ratio without maintaining original aspect ratio
* `rect:100,0,0,#:#,50`: Zoom to 100% [0-100] of the image, offset at 0%,0%, in the image's aspect ratio.
Then trim it to the specified aspect ratio and offset
* `image-frame:#x#x#x#,#color/name,true/false`: Apply an frame to the image of size # (or #x#, or #x#x#x#),
with the specified color or named image, above/below the image.

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

Image Settings Affecting Layout
-------------------------------
* `float:x,y,w,h`: - Displays an image that did not have space laid out for it
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
* `caption-position:below`: `above` or `below`
* `caption-squareness:100`: percentage of the weight of the actual image aspect ratio and square to give portrait images more space for captions.  0 means lay the caption out as if the image was square, and 100 means lay the caption out with the image's actual aspect ratio.

Text Settings
-------------
* `textAlign:left,right,center,justified,binding,edge` (binding & edge allowed if binding:side)
* `font:name`: font name
* `font-size:11`: font size in units
* `linespacing:1.2`: line spacing in percent
* `letterspacing:0`: letter spacing in units, positive or negative, real number
* `padding:0,0,0,0`: Text padding in the order Top, Right, Bottom, Left, or Top, Binding, Bottom, Edge for align:binding or align:edge
* `padding:0,0`: Text padding in the order Top/Bottom, Right/Left
* `padding:0`: Text padding, all the same
* `text-wrap:balanced,unbalanced`: do not break lines to make lines equal in length
* `width:`: width of text block in units, percent, or percent of remainder
* `float:#,#`: Displays text that did not have space laid out of it. Specifies the offset X and Y where to put the text.

Text or Image Settings
----------------------
* `name`: creates a named image or text that can be used for headers, footers, backgrounds, frames, etc. 
* `column-break:yes,no` (as a shortcut, may be specified without ":yes")
* `row-break:yes,no` (as a shortcut, may be specified without ":yes")
* `page-break:yes,no` (as a shortcut, may be specified without ":yes")
* `rotate`: after layout or float, only multiples of 90 are supported

## Command Line Options

With the exception of the input file name all command line options may be specified in the input file.  What is provided on the command line
takes precedence.

* `input-file`: Specify the input `.pb` file
* `--output-file`: Specify the output `.pdf`, `.png`, `.jpg`, or `.jpeg` file
* `--page-range`: Specify the pages to render
* `--verbose`: 
    * `D`: Details
    * `P`: Print processed input file
    * `X`: Print processed input file with comments
    * `L`: Logging
* `--watch`: Specify `1` to watch the input file and reprocess when it changes.
Otherwise, process the input file once and then exit
* `--cjpeg-command path-to-cjpeg`: When using mozjpeg to render, specify the path to the executable

### Developer Options
* `--cache-mode`: 
    * `1`: No image caching
    * `2`: Cache image information only while watching
    * `4`: Persist image cache and reuse always
    * `8`: No resize caching
    * `16`: Cache resiz information only while watching
    * `32`: Persist image cache and reuse always
* `--noresize`: Specify `1` to disable resizing step
* `--nolayout`: Specify `1` to disable layout step
* `--norender`: Specify `1` to disable rendering step

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

## Things to Do

* Output a text as an image (can do already, need to get height from verbose output)
* Input and output handlers for more file types
* Output sigmoidal brightness/lightness?  Existing output gamma is one type of this
* spreadedge, spreadbinding - just not implemented
* eyelevel, spreadeyelevel, mouthlevel, spreadmouthlevel - can do with blank texts as placeholders
* highlights, midtones, shadows
* Image, font https://... downloaded and then cached (in a zip file?)
* HSL Adjustment
* Named pages that could be embedded as an image in a page
* Calendar pages