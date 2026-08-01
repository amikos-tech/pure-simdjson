"""Strict YAML workflow loading shared by release contract tests."""

from __future__ import annotations

import json
import pathlib
import subprocess

try:
    import yaml
except ModuleNotFoundError:  # pragma: no cover - exercised via yq fallback when PyYAML is absent
    yaml = None


if yaml is not None:
    class UniqueKeyLoader(yaml.BaseLoader):
        pass


    def _construct_unique_mapping(
        loader: UniqueKeyLoader, node: yaml.nodes.MappingNode, deep: bool = False
    ) -> dict[object, object]:
        mapping: dict[object, object] = {}
        for key_node, value_node in node.value:
            key = loader.construct_object(key_node, deep=deep)
            if key in mapping:
                raise AssertionError(f"duplicate YAML key: {key!r}")
            mapping[key] = loader.construct_object(value_node, deep=deep)
        return mapping


    UniqueKeyLoader.add_constructor(
        yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG,
        _construct_unique_mapping,
    )


def load_workflow_definition(path: pathlib.Path, repo_root: pathlib.Path) -> dict[object, object]:
    workflow_text = path.read_text(encoding="utf-8")
    if yaml is not None:
        workflow = yaml.load(workflow_text, Loader=UniqueKeyLoader)
    else:
        result = subprocess.run(
            ["yq", "-o=json", str(path)],
            cwd=repo_root,
            check=False,
            capture_output=True,
            text=True,
        )
        if result.returncode != 0:
            raise AssertionError(f"unable to parse workflow YAML: {result.stderr.strip()}")
        workflow = json.loads(result.stdout)
    if not isinstance(workflow, dict):
        raise AssertionError(f"expected workflow mapping, got {type(workflow)!r}")
    return workflow
