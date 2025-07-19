endpoint name can be generateScene or something like that. 
The motion canvas tools should for now just:
Tool 1 :  a fetch svgs function that takes as input some query for the iconify api and downloads the svg icons from that api response back into a folder. here is some context about the iconify api, but idk if it i should git clone the repo and use the downloaders locally or host the api somewhere, and then run api requests to that api How to use icons in your projects?

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

Tool 2 : a fetch_examples, that creates a list of examples for the coding agent to have context. Currently, the best option for this would be to copy the from the context files in the frontend/src/scenes <example>.tsx files the example codes. We will make this more fancy as time goes on. 
Here is the current code i was using for this 

import yaml, json
from pathlib import Path
from typing import List, Type
from string import Template
from pydantic import BaseModel
from jinja2 import Template

def load_context_files(context_dirs: List[Path], extensions_allowed: List[str] = ["tsx"]) -> dict[str, str]:
    context_files = dict()
    for context_dir in context_dirs:
        assert context_dir.is_dir(), f"Context directory {context_dir} is not a directory"
        for extension in extensions_allowed:
            for file in context_dir.glob(f"*.{extension}"):
                print(file)
                context_files[file.name] = Path(file).read_text()
    return context_files

def load_yaml_template(yaml_path: Path) -> dict:
    with yaml_path.open() as f:
        return yaml.safe_load(f)

def render_examples(examples: list[dict]) -> str:
    blocks = []
    for ex in examples:
        input_ = ex.get("input", "")
        output = ex.get("output", "")
        review = ex.get("review")
        block = [f"INPUT: {input_}", "OUTPUT:", output]
        if review:
            block += ["REVIEW:", yaml.safe_dump(review).strip()]
        blocks.append("\n".join(block))
    return "\n\n".join(blocks)

def render_context(context_files: dict[str, str]) -> str:
    if not context_files:
        return ""
    parts = [f"FILE: {name}\n{content}" for name, content in context_files.items()]
    return "\n\n".join(parts)

def build_system_prompt(template_str: str, output_schema: str, examples: str, context: str) -> str:
    template = Template(template_str)
    return template.render(
        output_template=output_schema,
        examples=examples,
        context=context,
    )

def build_system_prompt_from_dirs_and_yaml(context_dirs: List[Path], yaml_path: Path, output_model: Type[BaseModel]):
    context_files = load_context_files(context_dirs)
    yaml_data = load_yaml_template(yaml_path)

    system_base = yaml_data.get("system_prompt", "")
    examples = yaml_data.get("examples", [])
    output_schema = json.dumps(output_model.model_json_schema(), indent=2)

    examples_str = render_examples(examples)
    context_str = render_context(context_files)

    final_prompt = build_system_prompt(
        template_str=system_base,
        output_schema=output_schema,
        examples=examples_str,
        context=context_str,
    )
    return final_prompt
if __name__ == "__main__":
    context_dirs = [Path("frontend/src/scenes"),Path("frontend/src/scenes2")]
    yaml_path = Path("agents/prompts/coder_general_01.yaml")
    from agents.output_models.code_output import CodeOutput
    final_prompt = build_system_prompt_from_dirs_and_yaml(context_dirs, yaml_path, CodeOutput)
    print(final_prompt)

Those are the two tool I plan to use. 
But right now I would like to focus on implementing the agent without these tools