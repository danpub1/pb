# pb

pb is a text-to-photo book conversion tool for photographers.
It allows focusing on the content of a book rather than layout and formatting.
The first design goal is to be able to simply list photos interspersed with text,
and convert that to a nicely formatted photo book.
But a photo book should also be a nice finished product, so the second design goal
is to support enough formatting to make that possible. 

## Introduction

### Simple Images and Text

If a line  looks like it starts with the name of a image file name
(i.e. there is something that ends with .jpg, .jpeg, .tif, .tiff, .bmp or .png), it should be treated as such and the image
laid out on the current page.

If the whole line does not look like the name of an image file name, 
then the first part should treated as an image file name and the image is laid out on the current page,
and the text following the image file name should be treated as a caption for the image.

If a line does not look like an image file name, it should be treated as text to be included. 

    Old Faithful.jpeg
    Steamboat Geyser.jpg
    Two of the famous geysers in Yellowstone.
    Old-Faithful.jpeg The most famous geyser!

In case there is any ambiguity, a line starting with an exclamation point followed by a space should be treated as an image.
If the file name contains spaces or single quotes, it should be wrapped in single quotes to remove ambiguity.  Any
single quotes in the file name would then need to be doubled. 

Similarly, a line starting with a hash or pound sign (#) followed by a space
should be treated as a line of text.

    ! 'Old Faithful''s Erupting.jpeg'
    ! Steamboat Geyser.jpg
    # Two of the famous geysers in Yellowstone.

### Wildcards

Multiple images can be specified with wildcards (asterisks and question marks).  All images matching the names will be included. 

    ! Vacation*.jpg

### Captions

Normally, a block of text goes all the way across the page. To make a text that corresponds to an image, include it on the same line:

    'Steamboat Geyser.jpg' Steamboat Geyser is one of the famous ones in Yellowstone.
    ! 'Old Faithful''s Erupting.jpeg' # Old Faithful's Erupting!

### Blank Lines 

To make the text file easier to read, blank lines may be included and they should be ignored.
Any line that begins with one or more spaces should be treated as a continuation of the previous line.

### Comments

Any line beginning with two forward slashes should be ignored, so it can be used for notes.

    Old Faithful.jpeg
    Steamboat Geyser.jpg

    Two of the famous geysers in Yellowstone.
       I saw both of these erupting during my visit.
    // TODO: go back and figure out what the dates were!

    Old-Faithful.jpeg The most famous geyser!

## Directives

`***` = book  
`+++` = page  
`---` = row  
`$` = style  
`!` = image  
`#` = text  
`>` = child  
`@` = include  
`|` = column

Given a list of images and text, pb will arrange them into rows and pages. How it does that can be controlled
with directives, which are additional lines starting with -, --, ---, etc.

    *** size:621x810 margins:45,45,45,54

    ! 'Old Faithful''s Erupting.jpeg'
    ! Steamboat Geyser.jpg
    # Two of the famous geysers in Yellowstone.

Directives have settings (in the example above, size and margins are the settings), and settings have names and values 
(the value of the size setting above is 621x810).  Settings have default values, which are used if the directive is not
specified.  For example, the setting 'units' has a default value of 'pt' (points).  Since it was not specified in the book directive,
the size is assumed to be in points, 621 points wide by 810 points tall (or 8.625 x 11.25 inches).  If a value 
needs to have a space, wrap it in single quotes like you would with an image name.

Images and text have directives also, ! and #, respectively, so that the following is equivalent to 
the previous example:

    *** size:621x810
          margins:54,45,45,45
    ! image:'Old Faithful''s Erupting.jpeg'
    ! image:'Steamboat Geyser.jpg'
    # text:'Two of the famous geysers in Yellowstone.'

Like other directives, images can have settings (a full list will follow).  The following are equivalent to each other:

    ! image:'Old Faithful''s Erupting.jpeg' straighten:1 frame:4,#FF0000
    'Old Faithful''s Erupting.jpeg' straighten:1 frame:4,#FF0000

Text can also have settings.  The following are equivalent:

    # content:'Two of the famous geysers in Yellowstone.' padding:1
    # padding:1 Two of the famous geysers in Yellowstone.

Note that without an explicit 'image' setting, image settings come after the image name.  Without an explicit 'text' setting,
text settings come before the text.

## Styles

Most of the settings for text are specified via a style, which can be applied in three ways:

1. {{style-name}}
1. Starting the text with #<number\>
1. Starting the text with multiple #'s

The following lines are all equivalent:

    # {{3}} Two of the famous geysers in Yellowstone.
    #3 Two of the famous geysers in Yellowstone.
    ### Two of the famous geysers in Yellowstone.

Styles with a number for the name can be applied right in the leading #.  

Alignment may also be applied in the leading #, by appending:

* C for Center
* R for Right
* L for Left
* J for Justified
* B for Binding
* E for Edge

The following lines are all equivalent:

    # {{3}} align:center Two of the famous geysers in Yellowstone.
    #3C Two of the famous geysers in Yellowstone.
    ###C Two of the famous geysers in Yellowstone.

Text without a leading # uses style 1 and the default alignment.  A typical setup is to use style 1 for body text, 
style 2 for sub headings, and style 3 for main-headings.

Styles are defined using $ followed by a space, the name of the style, and the settings being defined:

    $ 1 align:center font:'Fira Code Medium'

Although particularly useful with text, styles can be defined and applied to other directives also.
The easiest way to understand styles is that {{name}} is simply replaced with everything that was defined for it.
Similar replacement is made within a text.

    $ title Yellowstone Vacation Photos

## Special Style Names

There are several pre-defined styles, useful as part of texts in headers and footers:

{{Date}}
{{Year}}
{{FileName}}
{{PageNumber}}
{{TotalPages}}

## Special Texts

A text with a setting `name` is not part of the layout, but is used for various purposes, such as header and footer text:

    #1E name:header-first {{title}}\t{{Date}}\t{{PageNumber}}
    #1E name:footer-first Copyright {{Year}}, All rights reserved.

The predefined names are `header-` and `footer-` along with `first`, `last`, `even`, `odd`, and/or `any`.

## Tabs

`\t` in a text is a tab.  Two tabs are pre-defined: the first is a center tab, and the second is a right tab.
Prefixing a text with one tab is the equivalent of `align:center`, and prefixing with two tabs is the equivalent of `align:right`.
However, as shown in the example above, tabs are most useful when there is text before the tab.

## Special Images

An image with a setting `name` is not part of the layout, but is used for a page background or image frame.

## Nested Layout

TODO

A nested layout is a set of lines starting with >.
The first line in the section must either specify the aspect ratio and size settings, 
or size and offset settings to specify a floating
layout.  Page-level directives in a nested layout only
apply to the nested layout.

When an image and a text are specified on the same line together
(or, equivalently, when an image has a content setting),
this is considered a caption.
Captions are treated as a nested layout,
where the aspect ratio of the layout is calculated based on the aspect 
ratio of the image plus a pre-defined number of caption lines,
plus gutters.
So the following are roughly equivalent:

    ! Old Faithful's Erupting.jpeg # Old Faithful's Erupting!
    
    | 50% 
    > Old Faithful's Erupting.jpeg
    > Old Faithful's Erupting!

# Settings

Setting values that are indicated as `yes` or `no` may equivalently be `on` or `off`, `true` or `false`.

Setting values that are colors may be specified as:

* \#A - Equivalent to #AA, i.e. a gray of value AA
* \#BC - Equivalent to #BCBCBC, i.e. a gray of value BC
* \#ABC - Equivalent to #AABBCC
* \#ABCD - Equivalent to #AABBCCDD, 
* \#ABCDEF - An RGB triple with 100% opacity
* \#89ABCDEF - An RGB triple with opacity EF

Book-Level Settings
-------------------

The following settings only apply at the book level.

* `units:pt`: the units of measure used in laying out the book.  One of `in`, `cm`, `mm`, `pt`
* `density:2`: pixels per unit when converting the content to a final bitmap.  2 pixels per pt (144 ppi) could be considered for a preview quality, and 5 pixels per pt (360 ppi) could be appropriate for printing.
* `output-gamma:1.0`: apply a gamma correction to the final output. This is useful to lighten or darken printed output so it better matches the onscreen experience.
* `output-sharpen:4,1,0.5,0`: apply octave sharpening to the final bitmap after resizing is complete.
* `output-mozjpeg:yes`: use the mozjpeg compressor to create the final bitmap.
* `output-compression:97`: define the jpeg compression level when creating the final bitmap
* `binding:side`: define the binding location, one of `side`, `top`, `none`.  Controls if margins are alternated by even/odd pages, and how the UI displays spreads
* `background-first`
* `background-even`
* `background-odd`
* `background-last`
* `background-any` ?
* `filename`

Book- or Page-Level Settings
----------------------------
* `size:576x576`: Page size, width x height, in units.
* `margin:0,0,0,0`: Margins in the order Top, Right, Bottom, Left (binding:none), Top, Binding, Bottom, Edge (for binding:side), or Binding Right Edge Left (binding:top).  In units or percent.
* `margin:0,0`: Margins in the order Top/Bottom, Right/Left
* `margin:0`: Margins, all the same
* `textAlign:left`: Alignment of non-specified text, equivalent to `#L`
* `background:color:#F,image:name,stretch:no,trim:x%,fit:x%,ninepatch:border,gradient:#F-#F`

QUESTION
--------
* `background:tile`

Book-, Page-, or Nested Page-Level Settings
-------------------------------------------
* `minsize:0`: Minimum size in units of any dimension of image, used during layout
* `gutter:0,0`, `gutter:0%,0%`: vertical space (between rows), horizontal space (between images) (or percent of page)
* `valign:spreadmiddle`: vertical spacing of rows in a layout.  Specifies how extra space is distributed after rows are laid out.  One of:
  * `middle`: gutter is placed between rows, any extra is divided equally between the top and bottom
  * `justify`: extra is evenly distributed between rows
  * `top`: extra is placed at the bottom
  * `bottom`: extra is placed at the top
  * `binding`: extra is placed at the binding edge (if binding is top)
  * `edge`: extra is placed opposite the binding edge (if binding is top)
  * `spreadtop`: extra is distributed between rows and at the bottom
  * `spreadbottom`: extra is distributed between rows and at the top
  * `spreadmiddle`: extra is distributed between rows and at the top and bottom

QUESTION
--------
* `minsize:0%`: Minimum size in percentage of shorter dimension

Book-, Page-, Nested Page-, or Row-Level Settings
-------------------------------------------------
* `minsize:0`, `minsize:0%`: Minimum size in units (or in percent of shorter dimension of page) of any dimension of image, used during layout
* `row-weight:1`: Make rows bigger to fill the page.
* `halign:spreadcenter`: horizontal spacing of images in a row.  Specifies how extra space is distributed after images are laid out.
  * `center`: gutter is placed between images, any extra is divided equally between the left and right
  * `justify`: extra is evenly distributed between images
  * `left`: extra is placed at the right
  * `right`: extra is placed at the left
  * `binding`: extra is placed at the binding edge (if binding is side)
  * `edge`: extra is placed opposite the binding edge (if binding is side)
  * `spreadleft`: extra is distributed between pictures and at the right
  * `spreadright`: extra is distributed between pictures and at the left
  * `spreadcenter`: extra is distributed between pictures and at the right and left

Image-Level Settings
--------------------
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

Image-Level Settings Affecting Layout
-------------------------------------
* `weight:1` - when laying out, resize this image to use this proportion of remaining space 
* `name`: - image can be referred to for frames, not included in layout or floated
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

Book-, Page-, Nested Page-, Row-, or Image-Level Settings
---------------------------------------------------------
* `caption-position:below`: `above` or `below`
* `caption-squareness:100`: percentage of the weight of the actual image aspect ratio and square to give portrait images more space for captions.  0 means lay the caption out as if the image was square, and 100 means lay the caption out with the image's actual aspect ratio.
* `sorted-by:name`: identifies the sort order for images with wildcards
  * `name`
  * `date`
  
Text-Level Settings
-------------------
* `textAlign:left,right,center,justified,binding,edge` (binding & edge allowed if binding:side)
* `font:name`: font name
* `size:11`: font size in units
* `linespacing:120`: line spacing in percent
* `letterspacing:0`: letter spacing in units, positive or negative, real number
* `padding:0,0,0,0`: Text padding in the order Top, Right, Bottom, Left, or Top, Binding, Bottom, Edge for align:binding or align:edge
* `padding:0,0`: Text padding in the order Top/Bottom, Right/Left
* `padding:0`: Text padding, all the same
* `text-wrap:balanced,unbalanced`: do not break lines to make lines equal in length
* `width:`: width of text block in units, percent, or percent of remainder

Text- or Image-Level Settings
-----------------------------
* `float:WxH+X+Y` - takes image / text out of layout
* `rotate`: after layout or float


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