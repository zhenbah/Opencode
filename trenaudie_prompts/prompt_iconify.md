The motion canvas tools should for now just be a fetch svgs, that takes as input some query for the iconify api and downloads the svg icons from that api response back into a folder. here is some context about the iconify api, but idk if it i should git clone the repo and use the downloaders locally or host the api somewhere, and then run api requests to that api How to use icons in your projects?

Iconify ecosystem offers many ways to use icons, for both coders and designers.

HTML
For using icons in HTML, there are several viable options:

You can add icons to CSS.
You can add SVG to HTML.
Iconify offers unique components that render icons on demand.
SVG in CSS
How to use icons in CSS:

Add icon as a background or mask image in CSS.
Use <span> element in HTML to render it.
Using icons as background images works great for icons with hardcoded palette, such as emojis.

Using icons as mask images, in combination with setting background color to currentColor, allows using monotone icons in CSS. To change icon color, simply change text color.

Example showing icons used as background and mask images (hover to see color change):

See how to use icons in CSS for various tools and plug-ins that make it easy to add icons to CSS.

SVG in HTML
Icons can be embedded in HTML as <svg> elements:

svg
<svg xmlns="http://www.w3.org/2000/svg" width="1em" height="1em" viewBox="0 0 24 24">
   <path fill="currentColor" d="M12 4a4 4 0 0 1 4 4a4 4 0 0 1-4 4a4 4 0 0 1-4-4a4 4 0 0 1 4-4m0 10c4.42 0 8 1.79 8 4v2H4v-2c0-2.21 3.58-4 8-4Z"/>
</svg>
See how to add SVG to HTML for various tools and components that make it easy to add icons to HTML.

Icons on demand
Iconify ecosystem has a unique feature: Iconify API.

It is used by various icon components to load icon data on demand. Iconify icon components only load icon data for icons used on the page visitor is viewing, at run time, instead of bundling icons.

Iconify icon components are perfect for complex projects like theme or website customisers, customisable admin panels or any similar projects, where icons can be customised by user.

Iconify icon components are very easy to use. All a developer has to specify is an icon name:

html
<iconify-icon icon="mdi:home"></iconify-icon>
See how to use Iconify icon components.

Design
For designers, Iconify ecosystem offers several ways to easily import icons in various design tools.

Available options:

The designers who use Figma can install Iconify plug-in for Figma.
The designers who use Sketch users can install Iconify plugin-in for Sketch.
For other design tools, or if you are experiencing issues with plug-ins listed above, you can copy and paste SVG from one of the sources listed below.

