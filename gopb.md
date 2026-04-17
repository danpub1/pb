# pb

pb is a text-to-photobook conversion tool.
From a simple list of photos interspersed with text, it generates a reasonably formatted photobook. Its markup language supports sufficient formatting to make a nicely finished product.

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
* `binding:side`: define the binding location, one of `side`, `top`, `none`.  Controls if margins are alternated by even/odd pages
* `output-gamma:1.0`: apply a gamma correction to the page bitmap. This is useful to lighten or darken printed output so it better matches the onscreen experience.
* `output-sharpen:0.0`: apply octave sharpening to the page bitmap after resizing is complete.
* `output-compression:92`: define the jpeg compression level when creating the page bitmap
* `output-mozjpeg:false`: use the mozjpeg compressor to create the page bitmap.
* `output-mozjpeg-sampling:1x1`: specify the subsampling used by mozjpeg

Page-Level Settings
-------------------

* `size:576x576`: Page size, width x height, in units.
* `margin:0.0x0.0x0.0x0.0`: Margins in the order Top, Right, Bottom, Left (binding:none), Top, Binding, Bottom, Edge (for binding:side), or Binding Right Edge Left (binding:top).  In units or percent.
* `margin:0.0x0.0`: Margins in the order Top/Bottom, Right/Left
* `margin:0.0`: Margins, all the same
* `background:#color/name`: Color or named image to use as the page background
* `header:even-header,odd-header,offset,leading-pages,trailing-pages`: Named text to use as the header
* `footer:even-footer,odd-footer,offset,leading-pages,trailing-pages`:  Named text to use as the header
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
* `keep-columns-together`: false

Image Settings
--------------
* `straighten` degrees (Rotate Image)
* `brightness`: Amount
* `contrast`: Amount
* `gamma`: Amount
* `saturation`: Amount
* `scontrast`: sigmoidal contrast Amount, Knee
* `ssaturation`: sigmoidal saturation Amount, Knee
* `crop:WxH+X+Y` or `crop:WxH+X%+Y%`
* `trim:#:#,#%`: Trim & Crop to fit in aspect ratio (Center, Left/Top, Right/Bottom, 0-100% [50%])
* `fit:#:#,#%`: Shrink to fit in aspect ratio (Center, Left/Top, Right/Bottom, 0-100% [50%]))
* `frame:size:#,name:image,color:#F,border-size:#px,ninepatch:#px`

Image Settings Affecting Layout
-------------------------------
* `size` - indicates size before Layout
  * `normal` - minimum size (default) = scale:1
  * `larger` - minimum size * 1.25 = scale:1.25
  * `much-larger` minimum size * 1.25 * 1.25 = minimum size * 1.5625 = scale:1.5625
  * `smaller` - minimum size / 1.25 = minimum size * 0.8 = scale:0.8
  * `much-smaller` minimum size / 1.25 / 1.25 =  = minimum size / 1.5625 = minimum size * 0.64 = scale:0.64
  * `scale:#` - minimum size * #
  * `width:#` - specific width
  * `width:#%` - % of total width
  * `width:#%%` - % of remaining width
  * `height:#` - specific height
  * `height:#%` - % of total height
  * `height:#%%` - % of remaining height
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

Text or Image Settings
----------------------
* `name`
* `column-break:yes,no`
* `row-break:yes,no`
* `page-break:yes,no`
* `float:WxH+X+Y` - takes image / text out of layout
* `rotate`: after layout or float
* `page-break`: false

## Design

Functions in a pipeline

```
const (
    ItemTypeBook = iota
    ItemTypePage
    ItemTypeRow
    ItemTypeImage
    ItemTypeText
    ItemTypeGroup
)

type item struct {
   itemType int
   settings map[string]string
}
```

* Read(fileName string) ([]string) 
    * read file into a []byte, 
    * cast to string, 
    * split across line breaks to []string,
    * Join continued lines (with leading space)
    * filter out blank lines & comment lines
    * filter out leading hash-bang line
    * process include directives
    * Prefix unprefixed lines based on whether they appear to be image or text
    * Expand paths and wildcards in image and font lines 

* Canonicalize(lines []string) ([]item)
    * Create item based on type
    * Convert image + caption into column
    * Convert directives prefixed with > to children of parent item
    * parse setting:value to settings

* CalculateSizes(items []item) ([]item)
    * lookup aspect ratio, minimum width, minimum height for every image item
    * calculate height for every text item
    * calculate minimum height, minimum width for every image item with a caption, 
    * cache wherever possible

* Paginate(items []item) ([]item)
    * move image and text items into children members of row items, output []item
        * Calculate height and width of each row
    * move row items into children members of page items

* LayoutPages(items []item) ([]item)
    * Layout pages, assign each image and text item specific rectangles on the page

* OutputPages(items []item)
    * pb backend
    * GMIC backend
    * PDF backend
    * ImageMagick backend
    * SVG backend
    * LibreOffice backend