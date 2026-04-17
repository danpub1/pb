*** page-size:612x792 text-wrap:unbalanced distribute-rows:top

/// Define paths before using them
$$$ pz docsearls.zip::
$$$ fz Fira.zip::

$$$ 1 font:{{fz}}FiraSans-Light.otf,{{fz}}FiraMono-Regular.otf font-size:10
$$$ 2 font:{{fz}}FiraMono-Regular.otf font-size:8
$$$ 3 font:{{fz}}FiraSans-Bold.otf font-size:12
$$$ spacer $ padding:6 font-size:0 # .

>>> o example/example.pdf
>>> p -

$ font:{{fz}}FiraSans-Bold.otf font-size:24 linespacing:1.5 #C Examples

### Example 1: Pictures and text separate

## {{pz}}2011_07_09_dolomites_168.jpg
  \n{{pz}}2011_07_09_dolomites_204.jpg 
  \n#C Ra Gusela, at Passo Giau (Giau Pass). Shot while touring through Cortina and Italy's Dolomites,
  \n \0 \0 \0 in July 2021.\\nRa Gusela is in the Nuvolau Group of mountains in the Dolomites and in the
  \n \0 \0 \0 Veneto region of the country.

{{pz}}2011_07_09_dolomites_168.jpg
{{pz}}2011_07_09_dolomites_204.jpg 
#C Ra Gusela, at Passo Giau (Giau Pass). Shot while touring through Cortina and Italy's Dolomites, in July 2021.\nRa Gusela is in the Nuvolau Group of mountains in the Dolomites and in the Veneto region of the country.

/// # \nHello\nHello\\nHello\\\nHello\\\\nHello
/// # Hello\tHello\tHello
/// # \tHello\tHello
/// # \t\tHello
/// # Hello\t\tHello
/// # \1Hello\2Hello\3Hello\4Hello
/// # \a\b\c\d\e\f

{{spacer}}

### Example 2: Pictures with captions

## {{pz}}2021_04_13_waimea-canyon_035.jpg #C One of the amazing number of wild chickens in Kauai.
  \n \0 \0 \0 This one is at a stop by Waimea Canyon.
  \n{{pz}}2021_04_13_waimea-canyon_130.jpg #C Rocks on a ridge across from the Red Dirt Waterfall,
  \n \0 \0 \0 on the west edge of Waimea Canyon.

{{pz}}2021_04_13_waimea-canyon_035.jpg #C One of the amazing number of wild chickens in Kauai. This one is at a stop by Waimea Canyon.
{{pz}}2021_04_13_waimea-canyon_130.jpg #C Rocks on a ridge across from the Red Dirt Waterfall, on the west edge of Waimea Canyon.

+++

### Example 3: Picture with both caption and settings
## {{pz}}2021_04_13_waimea-canyon_135a.jpg $ rotate:90 #C The Red Dirt Waterfall,
  \n \0 \0 \0 on a ridge above Waimea Canyon.

{{pz}}2021_04_13_waimea-canyon_135a.jpg $ rotate:90 #C The Red Dirt Waterfall, on a ridge above Waimea Canyon.

{{spacer}}

### Example 4: Picture with settings but no caption
## {{pz}}2017_08_21_love-ranch_49.jpg $ brightness:25
{{pz}}2017_08_21_love-ranch_49.jpg $ brightness:25
/// #C Looking south across Love Ranch.

{{spacer}}

### Example 5: Text only with settings
## $ font-size:16 #C Looking south across Love Ranch.
$ font-size:16 #C Looking south across Love Ranch.

+++ size:33% distribute-rows:spreadmiddle

### Example 6: Settings in page directive, Wildcarded Image names

## +++ size:33%\n{{pz}}*.jpg # {{\0Filename}}

{{pz}}*.jpg # {{Filename}}

+++

# {{pz}}2016_07_06_niagara_255z.jpg\n
  Niagara Falls, illuminated at night.

# 2010_06_24_canaltrip_strasbourg_240z\n
  2010_06_24_canaltrip_strasbourg_266z\n
  2010_06_24_canaltrip_strasbourg_302\n
  The Strasbourg Cathedral. Built from 1015 to 1439, it was the tallest structure in the world, at 142m (466'), from 1647 to 1874. Located in Strasbourg, France.

# 2014_07_07_respect_001z\nSydney Opera House at dusk

# 2011_06_30_fco-bru_142\nThe Matterhorn. Zermatt is below the clouds behind it.
# 2011_06_30_fco-bru_150\nThe Matterhorn. Zermatt is below the clouds behind it. Kleine Matterhorn is on the right. The largest of the two lakes is Lago di Goillet. You can see the ski trails coming down from the Swiss side to the Italian (near) side, into Brueil-Cervinia.

##C 
 https://www.flickr.com/photos/docsearls/55094659211/in/album-72177720332025828
 https://www.flickr.com/photos/docsearls/51172938517/in/album-72157719209653980
 https://www.flickr.com/photos/docsearls/36618697611/in/album-72157685178733601
 https://www.flickr.com/photos/docsearls/27639230684/in/album-72157670857497086
 https://www.flickr.com/photos/docsearls/22631983307/in/album-72157658936194763
 https://www.flickr.com/photos/docsearls/14577658266/in/album-72157645517170246
 https://www.flickr.com/photos/docsearls/5888725281/in/album-72157626962564545