# Mapbot

Mapbot is a slack bot that provides tactical mapping to facilitate the play of
table-top role playing games within Slack- think Pathfinder or D&D. Mapbot
renders a square grid on top of user-provided images, provides tools to scale
the grid to line up with a map's existing grid, and allows the placement of
tokens on the map to represent characters and movement.

Mapbot is in its early stages, but has reached MVP- minimum viable product.
This means that your humble author believes mapbot is capable of providing
value and fulfilling its core purpose, though the are definitely some rough
edges, as-yet unimplemented features, and lurking bugs.

## How do I use it?

If you're interested in running your own instance of mapbot, please see "How
do I run it?" below. If you've already done that, or someone else has done it
for you, read on...

### Creating a map

Everything starts with you creating a map. Currently, mapbot requires you to provide a URL to the background image for your map. Google image search is a great way to find these, or you can create your own. If you're using a map you didn't create, please make sure you're respecting the artist's rights and staying within the licensing terms.

When you're creating your maps in mapbot, DMing mapbot is best; but you can also do it in a channel that mapbot has been invited to. The process goes like this:

* Create the map
* Size the grid
* Align the grid

#### Create the map

Creating the map is easy; pick a name, and tell mapbot `map add <name> <url>`.

(If you DM mapbot directly, you can send just that comment; if you're working in a channel, send `@mapbot map add <name> <url>`- the same command, prefixed with `@mapbot`.)

Maps are *yours*! Nobody else can use your maps unless you make them active in a channel, and the names you give your maps are yours alone.

![Map Add Screenshot](https://raw.githubusercontent.com/wiki/pdbogen/mapbot/mapbot-screen-add-map.png)

#### Sizing and Aligning the Grid

This is unfortunately the most painful part of the process at present, since it involves a lot of small changes and trial and error.

The grid overlay is required so that mapbot knows where to place tokens. We need to determine the image's DPI and the grid offset. The DPI is the number of pixels of the image that comprise an inch. This is basically determined on how "high resolution" the image is.

When you first create a map, the DPI will be set at the default of 10:

![Map 10 DPI Screenshot](https://raw.githubusercontent.com/wiki/pdbogen/mapbot/mapbot-screen-dpi-10.png)

The unfurled map view is fairly small, but clicking on the map will show the full resolution version:

![Map 10 DPI Zoom](https://raw.githubusercontent.com/wiki/pdbogen/mapbot/mapbot-zoom-dpi-10.png)

10 DPI is not fairly useful on this map, but if the actual DPI were closer to 10, we'd be able to count these small squares to arrive at an approximation of our actual DPI. Instead, let's try increasing DPI to 50.

![Map 50 DPI Screenshot](https://raw.githubusercontent.com/wiki/pdbogen/mapbot/mapbot-screen-set-dpi-50.png)

The zoomed in version of this 50dpi grid is very useful:

![Map 50 DPI Screenshot](https://raw.githubusercontent.com/wiki/pdbogen/mapbot/mapbot-zoom-dpi-50.png)

Simply by examining the top few squares, we can see that a map square is approximately 2.5 50DPI grid squares. This tells us our actual DPI is around 125- 50 * 2.5. To my eyes, it seemed just slightly less, so I next checked 124 DPI:

![Map 124 DPI Screenshot](https://raw.githubusercontent.com/wiki/pdbogen/mapbot/mapbot-zoom-dpi-124.png)

This is pretty close! But there's a problem- as our grid lines march down and to the right, we can see that they are increasingly out of alignment with the map. Specifically, the grid lines are moving UP and LEFT- indicating our DPI is too low. (DOWN and/or RIGHT would mean our grid was too high. DOWN and LEFT or UP and RIGHT would mean the map is not square.) Our grid squares are a little too short, and this error compounds as we move to the right.

It turns out this map was exactly 125 DPI:

![Map 125 DPI Screenshot](https://raw.githubusercontent.com/wiki/pdbogen/mapbot/mapbot-zoom-dpi-125.png)

_Even better_, it turns out that once we get the right DPI, this map's top-left corner is actually a map square, so we don't even need to adjust the offset.

Now that your map's grid is properly aligned, you can adjust the color (if you want):

![Map Grid Color Screenshot](https://raw.githubusercontent.com/wiki/pdbogen/mapbot/mapbot-screen-gridcolor.png)

...but otherwise, you're ready to use it!

#### Playing on a Map

Mapbot assumes play happens in the context of a channel- with you and other people. The first step is to select the active map, using the `map select` command:

![Map Select Screenshot](https://raw.githubusercontent.com/wiki/pdbogen/mapbot/mapbot-screen-select-map.png)

Once you've selected the map, you (and anyone else in the channel!) can add tokens to the map using the `token add` command:

![Map Place Token](https://raw.githubusercontent.com/wiki/pdbogen/mapbot/mapbot-screen-place-token.png)

Token names must be unique; but you can use either short words or emoji; an emoji token will be rendered in the full square.

## How do I run it?

Because mapbot uses a fair bit of CPU for its image operations and requires
persistent storage in the form a SQL database, it's not practical for the
author to provide mapbot to the public as a free service.

Instead, mapbot is designed for you to easily run your own; but this still
requires a bit of technical aptitude.

To run mapbot, you'll need the following:

* A mapbot "app" on Slack, which will let your instance of mapbot interact with slack teams. This "app" will be tied to a slack team, so you should consider creating your own.
* A PostgreSQL server; or an account on ElephantSQL. Mapbot can provision its own database via ElephantSQL, which provides a free tier.
* A server on which you can run mapbot; you can run this on Amazone's EC2 free tier, but the more horsepower you can provide, the better the experience will be.
* A domain name, since the process of adding mapbot to a slack team requires visiting a mapbot URL. Mapbot can use Let's Encrypt / ACME to obtain its own SSL certificate, or you can provide your own.

Most of these are free, but unfortunate it's a bit of work to get going. If
you're interested in donating to the author, he'd be interested in providing
mapbot as a public service.

Getting all of these set up is currently out of the scope of this document,
though a full tutorial may come later (and a pull request would be welcome).
Some information is included below about the parameters of the Slack app
required, but you're on your own for the rest. Please open a GitHub Issue if
you run into any problems.

### Slack App

Under "OAuth & Permissions", you'll need to configure a few things. The
Redirect URLs are URLs that Slack will return users to when they are adding
Mapbot to a new team. My URLs are:

    https://map.example.com
    http://localhost
    http://localhost:8080

The "localhost" URLs are useful for testing; and "map.example.com" is the
domain name on which I'm hosting mapbot.

Mapbot also requires several Permission Scopes to function:

  * chat:write:bot -- required since mapbot interacts via chat
  * bot -- as above
  * files:write:user -- mapbot uploads images of the map in response to map changes and token movements
  * team:read -- required to retrieve some metadata about the connected team, since data is stored per-team
  * emoji:read -- required to render your team's custom emoji onto the map as tokens

Under the "Bot Users" tab, make sure you've selected a reasonable default
username. In public channels, you'll interact with mapbot my starting messages
with `@<bot name>`, so pick something easy to type.

# Major Features

## Model

### Map

Maps are referred mostly as Tabula in code, since map is a reserved word in golang. :sobbing:

* [X] Background Image
* [X] Alignment (offset & DPI)
* [ ] Mask Sets
    * [ ] New
    * [ ] Add Rectangle
    * [ ] Clear Rectangle
    * [ ] Enable / Disable
* [ ] Overlays
* [X] Overlay coordinates

### Token

* [X] Tokens are per _context_, which is a combination of Team and Channel.

* [X] Name
* [X] Icon / Glyphicon
* [ ] Size
* [X] Position on map
* [X] Box color
* [ ] Stacking of small tokens

Commands:

* [X] List
* [X] Add
* [X] Set color
* [ ] Remove
* [ ] Clear
* [ ] Move <direction list>

## Slack UI

* [X] Add Map by URL
    * [X] Set map alignment
    * [X] Save map
    * [ ] Add by upload
* [X] Select Saved Map
* [ ] Add Mask Set
* [ ] Add rectangular mask to mask set
* [ ] Select/Unselect mask  set
* [ ] Spell effect overlay
* [ ] Add character ("add me")
* [ ] Move character
    * [ ] Cardinal directions
    * [ ] "Move To"

## Web UI

* [ ] Set map alignment GUI
* [ ] Complex mask set

# License

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program.  If not, see <http://www.gnu.org/licenses/>.

## Fonts

This package includes fonts from the Deja Vu font package. These fonts are available and distributed under the following license:

Fonts are © Bitstream (see below). DejaVu changes are in public domain. Explanation of copyright is on [Gnome page on Bitstream Vera fonts](http://gnome.org/fonts/). Glyphs imported from [Arev fonts](http://dejavu-fonts.org/wiki/Bitstream_Vera_derivatives#Arev_Fonts) are © Tavmjung Bah (see below)

### Bitstream Vera Fonts Copyright

Copyright (c) 2003 by Bitstream, Inc. All Rights Reserved. Bitstream Vera is a trademark of Bitstream, Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy of the fonts accompanying this license ("Fonts") and associated documentation files (the "Font Software"), to reproduce and distribute the Font Software, including without limitation the rights to use, copy, merge, publish, distribute, and/or sell copies of the Font Software, and to permit persons to whom the Font Software is furnished to do so, subject to the following conditions:

The above copyright and trademark notices and this permission notice shall be included in all copies of one or more of the Font Software typefaces.

The Font Software may be modified, altered, or added to, and in particular the designs of glyphs or characters in the Fonts may be modified and additional glyphs or characters may be added to the Fonts, only if the fonts are renamed to names not containing either the words "Bitstream" or the word "Vera".

This License becomes null and void to the extent applicable to Fonts or Font Software that has been modified and is distributed under the "Bitstream Vera" names.

The Font Software may be sold as part of a larger software package but no copy of one or more of the Font Software typefaces may be sold by itself.

THE FONT SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO ANY WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT OF COPYRIGHT, PATENT, TRADEMARK, OR OTHER RIGHT. IN NO EVENT SHALL BITSTREAM OR THE GNOME FOUNDATION BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, INCLUDING ANY GENERAL, SPECIAL, INDIRECT, INCIDENTAL, OR CONSEQUENTIAL DAMAGES, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF THE USE OR INABILITY TO USE THE FONT SOFTWARE OR FROM OTHER DEALINGS IN THE FONT SOFTWARE.

Except as contained in this notice, the names of Gnome, the Gnome Foundation, and Bitstream Inc., shall not be used in advertising or otherwise to promote the sale, use or other dealings in this Font Software without prior written authorization from the Gnome Foundation or Bitstream Inc., respectively. For further information, contact: fonts at gnome dot org.

### Arev Fonts Copyright

Original text

Copyright (c) 2006 by Tavmjong Bah. All Rights Reserved.

Permission is hereby granted, free of charge, to any person obtaining a copy of the fonts accompanying this license ("Fonts") and associated documentation files (the "Font Software"), to reproduce and distribute the modifications to the Bitstream Vera Font Software, including without limitation the rights to use, copy, merge, publish, distribute, and/or sell copies of the Font Software, and to permit persons to whom the Font Software is furnished to do so, subject to the following conditions:

The above copyright and trademark notices and this permission notice shall be included in all copies of one or more of the Font Software typefaces.

The Font Software may be modified, altered, or added to, and in particular the designs of glyphs or characters in the Fonts may be modified and additional glyphs or characters may be added to the Fonts, only if the fonts are renamed to names not containing either the words "Tavmjong Bah" or the word "Arev".

This License becomes null and void to the extent applicable to Fonts or Font Software that has been modified and is distributed under the "Tavmjong Bah Arev" names.

The Font Software may be sold as part of a larger software package but no copy of one or more of the Font Software typefaces may be sold by itself.

THE FONT SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO ANY WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT OF COPYRIGHT, PATENT, TRADEMARK, OR OTHER RIGHT. IN NO EVENT SHALL TAVMJONG BAH BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, INCLUDING ANY GENERAL, SPECIAL, INDIRECT, INCIDENTAL, OR CONSEQUENTIAL DAMAGES, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF THE USE OR INABILITY TO USE THE FONT SOFTWARE OR FROM OTHER DEALINGS IN THE FONT SOFTWARE.

Except as contained in this notice, the name of Tavmjong Bah shall not be used in advertising or otherwise to promote the sale, use or other dealings in this Font Software without prior written authorization from Tavmjong Bah. For further information, contact: tavmjong @ free . fr.

## Graphics

### Maps

MapBot uses maps you supply. You are responsible for ensuring that your use of maps is fair use or that you use maps under an appropriate license.

### Custom Emoji

MapBot uses your team's custom emoji. As with maps, ensure your use of any emoji is fair use or that you have an appropriate license.

### Standard Emoji

MapBot uses EmojiOne for standard emoji. Learn more at http://emojione.com/.
