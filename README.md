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

I'm experimenting with providing mapbot as a service. You can reach the running
instance of mapbot at https://mapbot.cernu.us and add it to your own Slack
team. Give it a try, and please let me know if you have any feedback.

After that, I recommend reading the following sections:

* [Creating a Map](#creating-a-map)
* [Aligning the Grid](#aligning-the-grid)
* [What Can Mapbot Do?](#what-can-mapbot-do)

### How do I run it myself?

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

#### Aligning the Grid

**New feature**: Mapbot can now guide you through the process of aligning a
grid to your map. This is intended to be much easier and much less
trial-and-error than the old, manual process. To get started, after you've
added a map, just use the `align` command to begin, like: `map add test
https://…` and then `map align test`.

As always, I welcome any feedback or suggestions around this process.

##### Manual Alignment

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

## What Can Mapbot Do?

New features are added from time to time; for the gory details, feel free to read the commit log. What follows is an overview of mapbot's major features.

### Move and Place Tokens

The heart of mapbot is the ability to move tokens around on a cartesian grid,
with a background image- a map. Once a map is made active in a channel (`map
select <map-name>`), any person in that channel can manipulate any of the
tokens on the map.

#### Adding and Moving Tokens

Adding and moving tokens uses the same command and the same process: `token
add <name> <location>[ <location2> … <locationN>]` or `token move <name> <location>` (these commands do
the exact same thing; the aliases help make interaction more natural.)

Examples:

* Add token `:simple_smile:` to the map at location A1: `token add
:simple_smile: a1`

* Move the token `:simple_smile:` to location Z10: `token add :simple_smile:
z10` or `token move :simple_smile: z10`

* Show a series of movements; from the starting location (implied, if the
token is on the current map), to x9, c9, c1, a1, in order): `token move
:simple_smile: x9 c9 c1 a1`

When moving a token, Mapbot will show a trail representing the token's path,
and calculate the distance moved according to Pathfinder rules.

**Advanced**: As a player, you typically only move around your own token.
Mapbot makes this easier- if you don't specify a token for an `add` or `move`
operation, it will effect whatever the last token you moved was: `token move
a1`

**Extra Advanced**: To help out GMs, you can add multiple tokens at once. Just
provide additional `<token name> <location>` pairs, like this example: `token
add foo a1 bar a2 fizz b3 buzz b4`

#### Removing Tokens

Removing a token is easy- `token remove <token_name>`. You can remove multiple
tokens at a time, too; just list out the names, like this example: `token
remove foo bar fizz buzz`

If you want to remove _all_ the tokens on the map, you can use the `clear`
sub-command: `token clear`.

Sometimes you want to _replace_ a token- maybe you mistyped the name, or maybe
there's a dramatic reveal! You can use the `swap` or `replace` sub-commands to
change the name of a token while keeping everything else the same: `token swap
foo bar`.

#### Token Special Effects

Here are some additional tips:

* If you want to use the same emoji multiple times (maybe you're fighting
three trolls?) you can append a string label after the emoji to differentiate
them, like this: `:troll:1` `:troll:2` `:troll:bob`. Make sure there are no
spaces!

* Need large, huge, giant tokens? No problem! Use `token size <name> <N>` to
tell mapbot that the token should take up multiple squares. The top-left
square is always treated as the main square, for moving tokens around and the
like.

* Need to represent auras or the effect of lights? Use `token light <name>
<radius>`. In fact, you can specify up to three radii- but be aware that the
second radius will conceal the first; and the third radius will conceal the
second. For example, `token light fizz 10 20 30` will only show the last, 30ft
radius! But `token light fizz 30 20 10` will show three concentric circles.

### Map Special Effects

Mapbot can do a few things on the map to help with your gameplay: Marks, which
can outline squares, draw lines, and indicate auras and areas-of-effect; and
Checks, which are exactly the same- but temporary.

#### Mark Squares

The easiest type of mark is marking a list of squares. Mark has pretty
flexible syntax, but we'll start easy:

* `mark a1 red` -- Marks the square at A1 red. By default, colors are
translucent, allowing you to see the map behind the color. You can always get
a solid version of the color by using `solidred` instead of `red`. You can
also use an HTML color code, like `#FF0000` for solid red; or `#FF0000C0` for
translucent red.

You can mark multiple squares at once, too:

* `mark a1 a2 b1 b2 red`

And you can even mark multiple squares different colors:

* `mark a1 red a2 green b1 b2 orange`

You can mark the edges or corners of squares by adding a cardinal direction to the coordinate:

* `mark a1ne red` marks the northeast corner of square a1 red. (This is also
the northwest corner of square b2, etc.)

* `mark a1e red` marks the entire eastern side of square a1 red. This is also
the western side of square b2.

You can _un_ mark a square by using the special color `clear`:

* `mark a1 clear`

Or you can unmark _every_ square by using the special _subcommand_ `clear`:

* `mark clear`

#### Marking Shapes

Instead of specifying one color at a time, you can also mark shapes- squares, circles, cones; and simple lines.

You can make squares by specifying the two corners:

* `mark square(a1,d4) red`

You can mark circles by specifying the center and a **radius**:

* `mark circle(c3,10) red`

In Pathfinder, circles often (but not always) originate at a grid intersection
instead of a square. Mapbot can calculate this for you, too. Just indicate
which corner fo the square is the center by adding `ne`, `se`, `sw`, or `nw`
(for northeast, southeast, southwest, or northwest) to the coordinate:

* `mark circle(c3se,10) red`

Cones are similar to circles, though they must start from a corner /
intersection instead of an entire square. You also need to specify the
direction, which can be any of `n`, `ne`, `e`, `se`, `s`, `sw`, `w`, or `nw`.

Mapbot should be able to draw arbitrarily-sized cones following the spirit of
the Pathfinder rules for cones; and the cones for sizes documented in the
books are guaranteed to match. But, if you identiy any problems, don't
hesitate to reach out!

* `mark cone(c3se,e,15) red`

To check cover or charge lanes, you can use the `lines` shape. Lines can be
drawn from a corner or a square, to another corner or square.

Draw lines from a corner to a square to check for ranged cover:

* `mark lines(a1se,f10) red`

Draw lines from a corner to a corner to check for line of effect / line of sight:

* `mark lines(a1se,f10nw) red`

Draw lines from a square to a square to check for a charge lane:

* `mark lines(a1,f10) red`

## How do I run it?

Mapbot is designed for you to easily run your own; but this still requires a
bit of technical aptitude.

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

Under the "Interactive Components" tab, enable Interactive Components and
provide an action URL of `https://<hostname>/action`.

# Major Features

## Model

### Map

Maps are referred mostly as Tabula in code, since map is a reserved word in golang. :sobbing:

* [X] Background Image
* [X] Alignment (offset & DPI)
    * [X] interactive / workflow-driven
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
* [X] Size
* [X] Position on map
* [X] Box color
* [ ] Stacking of small tokens
* [X] opened/closed door tokens _implemented via edge marks_

Commands:

* [X] List
* [X] Add
* [X] Set color
* [X] Remove
* [X] Clear
* [ ] Move <direction list>

## Slack UI

* [X] Add Map by URL
    * [X] Set map alignment
    * [X] Save map
    * [ ] Add by upload
* [X] Select Saved Map
* [ ] Add Mask Set
* [ ] Add rectangular mask to mask set
* [ ] Select/Unselect mask set
* [X] Spell effect overlay
    * [X] cones
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
