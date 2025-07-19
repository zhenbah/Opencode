
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