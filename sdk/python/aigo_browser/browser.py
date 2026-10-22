"""
AI Go Browser - Python Enterprise SDK
Ultra-low-footprint native semantic browser for AI agents and LLM pipelines.
"""

import json
import os
import platform
import subprocess
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional


@dataclass
class ButtonAction:
    ref: str
    label: str
    tag: str
    form_method: str = ""
    form_action: str = ""
    action_summary: str = ""
    hidden_inputs: Dict[str, str] = field(default_factory=dict)


@dataclass
class NavigationPage:
    url: str
    title: str
    markdown: str
    text: str
    status: int
    raw_bytes: int
    raw_tokens_est: int
    agent_tokens_est: int
    token_savings_pct: float
    rss_mb: float
    duration_ms: float
    links: List[Dict[str, str]] = field(default_factory=list)
    buttons: List[ButtonAction] = field(default_factory=list)
    forms: List[Dict[str, Any]] = field(default_factory=list)
    raw_response: Dict[str, Any] = field(default_factory=dict)


class Browser:
    """
    High-performance native browser client interfacing with the compiled Go engine.
    """

    def __init__(self, binary_path: Optional[str] = None):
        if binary_path and os.path.exists(binary_path):
            self.binary_path = binary_path
        else:
            # Look for local compiled binary in standard locations
            root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
            exe_ext = ".exe" if platform.system() == "Windows" else ""
            candidate = os.path.join(root_dir, "bin", f"aigobrowser{exe_ext}")
            if os.path.exists(candidate):
                self.binary_path = candidate
            else:
                self.binary_path = f"aigobrowser{exe_ext}"

    def navigate(
        self,
        url: str,
        compact: bool = False,
        format_output: str = "json",
        timeout_seconds: int = 20,
    ) -> NavigationPage:
        """
        Navigates to a target URL in pure Go native mode with Zero Chromium overhead.
        """
        cmd = [self.binary_path, "-format", format_output]
        if compact:
            cmd.append("-compact=true")
        cmd.append(url)

        try:
            res = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=timeout_seconds,
                check=True,
            )
            raw = json.loads(res.stdout)
        except subprocess.CalledProcessError as e:
            try:
                raw = json.loads(e.stdout)
            except Exception:
                raise RuntimeError(f"Navigation failed: {e.stderr or e.stdout}")
        except FileNotFoundError:
            raise FileNotFoundError(
                f"aigobrowser binary not found at '{self.binary_path}'. "
                f"Please compile with 'go build -o bin/aigobrowser ./cmd/aigobrowser' first."
            )

        if not raw.get("ok"):
            raise RuntimeError(raw.get("error", "Unknown browser error"))

        page_data = raw.get("page", {})
        memory_data = raw.get("memory", {})
        duration = float(raw.get("duration_ms", 0.0))

        # Parse buttons and action metadata
        buttons: List[ButtonAction] = []
        for b in page_data.get("buttons", []):
            act = b.get("action", {})
            buttons.append(
                ButtonAction(
                    ref=b.get("ref", ""),
                    label=b.get("label", ""),
                    tag=act.get("tag", "button"),
                    form_method=act.get("form_method", ""),
                    form_action=act.get("form_action", ""),
                    action_summary=act.get("action_summary", ""),
                    hidden_inputs=act.get("hidden_inputs", {}),
                )
            )

        raw_bytes = int(page_data.get("bytes", 0))
        raw_tokens = raw_bytes // 4 if raw_bytes > 0 else 0
        md_text = page_data.get("markdown", "") or page_data.get("text", "")
        agent_tokens = len(md_text) // 4
        savings = (
            (1.0 - (float(agent_tokens) / float(raw_tokens))) * 100.0
            if raw_tokens > 0
            else 0.0
        )

        return NavigationPage(
            url=page_data.get("url", url),
            title=page_data.get("title", ""),
            markdown=md_text,
            text=page_data.get("text", ""),
            status=int(page_data.get("status", 200)),
            raw_bytes=raw_bytes,
            raw_tokens_est=raw_tokens,
            agent_tokens_est=agent_tokens,
            token_savings_pct=max(0.0, savings),
            rss_mb=float(memory_data.get("rss_mib", 0.0)),
            duration_ms=duration,
            links=page_data.get("links", []),
            buttons=buttons,
            forms=page_data.get("forms", []),
            raw_response=raw,
        )
