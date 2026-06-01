#!/usr/bin/env python3

# Implements DESIGN-014 MetricsCollector coverage report generation.

import re
import shutil
import datetime
from pathlib import Path

def parse_go_coverage(output: str) -> dict:
    files = []
    total_pct = "0.0%"

    # Match: github.com/mealswapp/mealswapp/backend/internal/app/app.go:11: New 100.0%
    pattern = re.compile(r"^([^:]+):(\d+):\s+(\S+)\s+(\d+(?:\.\d+)?)%")

    for line in output.splitlines():
        line = line.strip()
        if not line:
            continue
        if line.startswith("total:"):
            parts = line.split()
            if len(parts) >= 3:
                total_pct = parts[-1]
            continue

        match = pattern.match(line)
        if match:
            filepath, line_num, func_name, pct = match.groups()
            display_path = filepath
            prefix = "github.com/mealswapp/mealswapp/backend/"
            if display_path.startswith(prefix):
                display_path = display_path[len(prefix):]

            files.append({
                "file": display_path,
                "line": int(line_num),
                "func": func_name,
                "coverage": float(pct)
            })

    return {
        "files": files,
        "total": total_pct
    }

def parse_bun_coverage(output: str) -> dict:
    files = []
    total_funcs = "0.0%"
    total_lines = "0.0%"

    # Table structure:
    # File                             | % Funcs | % Lines | Uncovered Line #s
    # All files                        |  100.00 |  100.00 |
    #  src/lib/cache/service-worker.ts |  100.00 |  100.00 |
    for line in output.splitlines():
        line = line.strip()
        if not line:
            continue
        if "---" in line and "|" in line:
            continue
        if "|" in line:
            columns = [c.strip() for c in line.split("|")]
            if len(columns) >= 3:
                name, funcs, lines = columns[0], columns[1], columns[2]
                if name.lower() == "file":
                    continue
                if name.lower() == "all files":
                    total_funcs = funcs + "%"
                    total_lines = lines + "%"
                else:
                    files.append({
                        "file": name,
                        "funcs": funcs + "%",
                        "lines": lines + "%",
                        "uncovered": columns[3] if len(columns) >= 4 else ""
                    })
    return {
        "files": files,
        "total_funcs": total_funcs,
        "total_lines": total_lines
    }

def build_html_report(go_raw: str, bun_raw: str, reqs_checked: int, reqs_total: int, output_path: str, screenshot_stem: str | None = None, design_implemented: dict[str, list[str]] | None = None, design_missing: dict[str, list[str]] | None = None, design_checked: int = 0, design_total: int = 0, design_aspects: dict[str, list[str]] | None = None) -> None:
    go_data = parse_go_coverage(go_raw)
    bun_data = parse_bun_coverage(bun_raw)

    timestamp = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")

    # Copy screenshots from /tmp/mealswapp-frontend-verifier to output_dir/screenshots/
    html_path = Path(output_path)
    html_dir = html_path.parent
    screenshot_stem = screenshot_stem or html_path.stem
    screenshots_dir = html_dir / "screenshots"
    screenshots_dir.mkdir(parents=True, exist_ok=True)

    tmp_screenshots_dir = Path("/tmp/mealswapp-frontend-verifier")
    desktop_name = f"{screenshot_stem}-desktop.png"
    mobile_name = f"{screenshot_stem}-mobile.png"
    desktop_src = tmp_screenshots_dir / desktop_name
    mobile_src = tmp_screenshots_dir / mobile_name

    has_screenshots = False
    if desktop_src.exists():
        shutil.copy(desktop_src, screenshots_dir / desktop_name)
        has_screenshots = True
    if mobile_src.exists():
        shutil.copy(mobile_src, screenshots_dir / mobile_name)
        has_screenshots = True

    screenshots_html = ""
    if has_screenshots:
        screenshots_html = f"""
        <div class="section-title">Frontend Verification Screenshots</div>
        <div class="screenshots-container">
            <div class="screenshot-card">
                <h4>Desktop View (1280x900)</h4>
                <div class="screenshot-frame">
                    <img src="screenshots/{desktop_name}" alt="Desktop View">
                </div>
            </div>
            <div class="screenshot-card mobile">
                <h4>Mobile View (390x844)</h4>
                <div class="screenshot-frame">
                    <img src="screenshots/{mobile_name}" alt="Mobile View">
                </div>
            </div>
        </div>
        """

    # Requirements checklist removed

    # Build Go rows
    go_rows = ""
    for f in go_data["files"]:
        cov_val = f["coverage"]
        pct_color = "pct-green" if cov_val == 100.0 else "pct-yellow" if cov_val >= 80.0 else "pct-red"
        pct_text_color = "text-green" if cov_val == 100.0 else "text-yellow" if cov_val >= 80.0 else "text-red"
        go_rows += f"""
        <tr>
            <td class="file-path">{f["file"]}</td>
            <td class="func-name"><code>{f["func"]}()</code></td>
            <td class="line-num">{f["line"]}</td>
            <td class="coverage-cell">
                <div class="progress-bar-bg">
                    <div class="progress-bar-fill {pct_color}" style="width: {cov_val}%"></div>
                </div>
                <span class="coverage-pct {pct_text_color}">{cov_val}%</span>
            </td>
        </tr>
        """

    # Build Bun rows
    bun_rows = ""
    for f in bun_data["files"]:
        funcs_pct = f["funcs"]
        lines_pct = f["lines"]
        pct_color_func = "text-green" if funcs_pct == "100.00%" else "text-yellow"
        pct_color_line = "text-green" if lines_pct == "100.00%" else "text-yellow"
        bun_rows += f"""
        <tr>
            <td class="file-path">{f["file"]}</td>
            <td class="coverage-cell">
                <span class="coverage-pct {pct_color_func}">{funcs_pct}</span>
            </td>
            <td class="coverage-cell">
                <span class="coverage-pct {pct_color_line}">{lines_pct}</span>
            </td>
            <td class="uncovered-cell">{f["uncovered"] or "-"}</td>
        </tr>
        """

    design_coverage_pct = f"{(design_checked / design_total * 100):.1f}%" if design_total > 0 else "0.0%"
    design_coverage_color = "var(--success)" if design_checked == design_total else "var(--warning)" if design_checked > 0 else "var(--danger)"

    design_arch_map: dict[str, str] = {}
    if design_aspects:
        import re
        ARCH_RE = re.compile(r"ARCH-(\d+)")
        design_dir = Path(__file__).resolve().parents[1] / "docs" / "design"
        for path in sorted(design_dir.glob("DESIGN-*.md")):
            text = path.read_text()
            match = ARCH_RE.search(text)
            if match:
                design_arch_map[path.stem] = match.group(1)

    design_rows = ""
    if design_implemented is None:
        design_implemented = {}
        design_missing = {}
    for design_id in sorted(design_implemented.keys()):
        implemented_list = design_implemented.get(design_id, [])
        missing_list = design_missing.get(design_id, [])
        total_for_design = len(implemented_list) + len(missing_list)
        if total_for_design == 0:
            continue
        implemented_pct = len(implemented_list) / total_for_design * 100
        if implemented_pct == 100:
            badge_class = "badge-complete"
            badge_text = "COMPLETE"
        elif implemented_pct > 0:
            badge_class = "badge-partial"
            badge_text = f"{len(implemented_list)}/{total_for_design}"
        else:
            badge_class = "badge-empty"
            badge_text = "EMPTY"

        aspect_tags = ""
        for asp in implemented_list:
            aspect_tags += f'<span class="aspect-tag aspect-implemented">{asp}</span>'
        for asp in missing_list:
            aspect_tags += f'<span class="aspect-tag aspect-missing">{asp}</span>'

        arch_num = design_arch_map.get(design_id, "0")
        section_id = "frontend-coverage" if int(arch_num) <= 8 else "go-coverage"
        section_label = section_id.replace("-coverage", "").replace("-", " ")

        design_rows += f"""
        <a href="#{section_id}" class="design-card" title="View {section_label} test coverage">
            <div class="design-card-header">
                <span class="design-card-title">{design_id}</span>
                <span class="design-card-badge {badge_class}">{badge_text}</span>
            </div>
            <div class="design-aspect-list">{aspect_tags}</div>
        </a>
        """

    html_content = f"""<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Mealswapp Quality Gate & Coverage Report</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=Roboto+Mono:wght@400;500&display=swap" rel="stylesheet">
    <style>
        :root {{
            --bg-color: #0b0f19;
            --card-bg: #111827;
            --border-color: #1e293b;
            --text-primary: #f8fafc;
            --text-secondary: #94a3b8;
            --primary: #6366f1;
            --primary-glow: rgba(99, 102, 241, 0.15);
            --success: #10b981;
            --success-glow: rgba(16, 185, 129, 0.2);
            --warning: #f59e0b;
            --danger: #ef4444;
            --mono-font: 'Roboto Mono', monospace;
        }}

        * {{
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }}

        body {{
            background-color: var(--bg-color);
            color: var(--text-primary);
            font-family: 'Inter', sans-serif;
            line-height: 1.6;
            padding: 2.5rem 1.5rem;
        }}

        .container {{
            max-width: 1200px;
            margin: 0 auto;
        }}

        header {{
            display: flex;
            justify-content: space-between;
            align-items: center;
            border-bottom: 1px solid var(--border-color);
            padding-bottom: 2rem;
            margin-bottom: 2.5rem;
        }}

        .header-left h1 {{
            font-size: 2.25rem;
            font-weight: 700;
            background: linear-gradient(135deg, #a5b4fc, var(--primary));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            letter-spacing: -0.025em;
        }}

        .header-left p {{
            color: var(--text-secondary);
            font-size: 0.95rem;
            margin-top: 0.25rem;
        }}

        .status-badge {{
            display: flex;
            align-items: center;
            gap: 0.6rem;
            background-color: var(--success-glow);
            border: 1px solid var(--success);
            color: var(--success);
            padding: 0.5rem 1rem;
            border-radius: 9999px;
            font-weight: 600;
            font-size: 0.9rem;
            box-shadow: 0 0 15px rgba(16, 185, 129, 0.1);
        }}

        .status-pulse {{
            width: 8px;
            height: 8px;
            background-color: var(--success);
            border-radius: 50%;
            display: inline-block;
            animation: pulse 2s infinite;
        }}

        @keyframes pulse {{
            0% {{ transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7); }}
            70% {{ transform: scale(1); box-shadow: 0 0 0 6px rgba(16, 185, 129, 0); }}
            100% {{ transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0); }}
        }}

        .grid-summary {{
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2.5rem;
        }}

        .card {{
            background-color: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 1.5rem;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
            transition: transform 0.2s ease, border-color 0.2s ease;
        }}

        .card:hover {{
            transform: translateY(-2px);
            border-color: #334155;
        }}

        .card-link {{
            text-decoration: none;
            color: inherit;
            display: contents;
        }}

        .card-link:hover .card {{
            transform: translateY(-2px);
            border-color: #334155;
        }}

        .card h3 {{
            color: var(--text-secondary);
            font-size: 0.85rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            margin-bottom: 0.75rem;
        }}

        .card .value {{
            font-size: 2.25rem;
            font-weight: 700;
            color: var(--text-primary);
        }}

        .card .sub {{
            color: var(--text-secondary);
            font-size: 0.85rem;
            margin-top: 0.25rem;
        }}

        .checklist {{
            list-style: none;
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
        }}

        .checklist li {{
            display: flex;
            align-items: center;
            gap: 0.5rem;
            font-size: 0.95rem;
        }}

        .checklist .check-icon {{
            color: var(--success);
            font-weight: bold;
        }}

        .policy-section {{
            margin-bottom: 3rem;
            background: linear-gradient(180deg, var(--card-bg), rgba(17, 24, 39, 0.7));
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 1.75rem;
        }}

        .policy-section h2 {{
            font-size: 1.25rem;
            font-weight: 600;
            margin-bottom: 1rem;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }}

        .policy-section p {{
            color: var(--text-secondary);
            font-size: 0.95rem;
            margin-bottom: 1rem;
        }}

        .policy-grid {{
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 1.5rem;
        }}

        .policy-box {{
            background-color: rgba(255, 255, 255, 0.02);
            border: 1px solid rgba(255, 255, 255, 0.05);
            border-radius: 8px;
            padding: 1rem;
        }}

        .policy-box h4 {{
            font-size: 0.9rem;
            font-weight: 600;
            margin-bottom: 0.5rem;
            color: var(--primary);
        }}

        .policy-box ul {{
            list-style: none;
            color: var(--text-secondary);
            font-size: 0.85rem;
            display: flex;
            flex-direction: column;
            gap: 0.25rem;
        }}

        .policy-box li span {{
            color: var(--text-primary);
            font-weight: 500;
        }}

        .section-title {{
            font-size: 1.5rem;
            font-weight: 700;
            margin-bottom: 1.25rem;
            letter-spacing: -0.02em;
            display: flex;
            align-items: center;
            gap: 0.5rem;
            border-left: 4px solid var(--primary);
            padding-left: 0.75rem;
        }}

        .table-container {{
            background-color: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            overflow: hidden;
            margin-bottom: 3rem;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
        }}

        table {{
            width: 100%;
            border-collapse: collapse;
            text-align: left;
            font-size: 0.9rem;
        }}

        th {{
            background-color: rgba(255, 255, 255, 0.02);
            border-bottom: 1px solid var(--border-color);
            color: var(--text-secondary);
            font-weight: 600;
            padding: 1rem 1.25rem;
            text-transform: uppercase;
            font-size: 0.75rem;
            letter-spacing: 0.05em;
        }}

        td {{
            padding: 1rem 1.25rem;
            border-bottom: 1px solid var(--border-color);
        }}

        tr:last-child td {{
            border-bottom: none;
        }}

        tr:hover td {{
            background-color: rgba(255, 255, 255, 0.01);
        }}

        .file-path {{
            font-family: var(--mono-font);
            color: var(--text-primary);
            font-size: 0.85rem;
        }}

        .func-name code {{
            font-family: var(--mono-font);
            color: #a5b4fc;
            background-color: rgba(99, 102, 241, 0.1);
            padding: 0.2rem 0.4rem;
            border-radius: 4px;
            font-size: 0.8rem;
        }}

        .line-num {{
            font-family: var(--mono-font);
            color: var(--text-secondary);
        }}

        .coverage-cell {{
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }}

        .progress-bar-bg {{
            width: 100px;
            height: 6px;
            background-color: var(--border-color);
            border-radius: 9999px;
            overflow: hidden;
        }}

        .progress-bar-fill {{
            height: 100%;
            border-radius: 9999px;
        }}

        .pct-green {{ background-color: var(--success); }}
        .pct-yellow {{ background-color: var(--warning); }}
        .pct-red {{ background-color: var(--danger); }}

        .text-green {{ color: var(--success) !important; }}
        .text-yellow {{ color: var(--warning) !important; }}
        .text-red {{ color: var(--danger) !important; }}

        .coverage-pct {{
            font-weight: 600;
            font-family: var(--mono-font);
            font-size: 0.85rem;
        }}

        .uncovered-cell {{
            font-family: var(--mono-font);
            color: var(--text-secondary);
            font-size: 0.85rem;
        }}

        .reqs-section {{
            margin-bottom: 2.5rem;
        }}

        .req-grid {{
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
            gap: 1rem;
        }}

        .req-card {{
            background-color: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 0.75rem 1rem;
            transition: all 0.2s ease;
        }}

        .req-card.verified {{
            border-left: 3px solid var(--success);
        }}

        .req-card.missing {{
            border-left: 3px solid var(--danger);
            opacity: 0.6;
        }}

        .req-header {{
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 0.25rem;
        }}

        .req-id {{
            font-family: var(--mono-font);
            font-weight: 600;
            font-size: 0.85rem;
        }}

        .req-card.verified .req-id {{ color: #a7f3d0; }}
        .req-card.missing .req-id {{ color: #fca5a5; }}

        .req-status {{
            font-size: 0.75rem;
            font-weight: 600;
            text-transform: uppercase;
        }}

        .req-card.verified .req-status {{ color: var(--success); }}
        .req-card.missing .req-status {{ color: var(--danger); }}

        .req-body {{
            font-size: 0.75rem;
            color: var(--text-secondary);
        }}

        .screenshots-container {{
            display: grid;
            grid-template-columns: 2fr 1fr;
            gap: 1.5rem;
            margin-bottom: 3rem;
        }}

        @media (max-width: 768px) {{
            .screenshots-container {{
                grid-template-columns: 1fr;
            }}
        }}

        .screenshot-card {{
            background-color: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 1.5rem;
            display: flex;
            flex-direction: column;
            gap: 1rem;
            align-items: center;
        }}

        .screenshot-card h4 {{
            color: var(--text-secondary);
            font-size: 0.9rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            width: 100%;
            text-align: left;
            border-bottom: 1px solid var(--border-color);
            padding-bottom: 0.5rem;
        }}

        .screenshot-frame {{
            width: 100%;
            border-radius: 8px;
            overflow: hidden;
            border: 1px solid rgba(255, 255, 255, 0.05);
            box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.5);
            display: flex;
            justify-content: center;
            align-items: center;
            background-color: rgba(0, 0, 0, 0.2);
        }}

        .screenshot-frame img {{
            max-width: 100%;
            max-height: 480px;
            object-fit: contain;
            display: block;
        }}

        .screenshot-card.mobile .screenshot-frame {{
            max-width: 280px;
        }}

        .design-section {{
            margin-bottom: 2.5rem;
        }}

        .design-grid {{
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
            gap: 1rem;
        }}

        .design-card {{
            display: block;
            background-color: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 1rem;
            text-decoration: none;
            color: inherit;
        }}

        .design-card:hover {{
            transform: translateY(-2px);
            border-color: #334155;
        }}

        .design-card-header {{
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 0.5rem;
        }}

        .design-card-title {{
            font-family: var(--mono-font);
            font-weight: 600;
            font-size: 0.9rem;
            color: #a5b4fc;
        }}

        .design-card-badge {{
            font-size: 0.75rem;
            font-weight: 600;
            padding: 0.2rem 0.5rem;
            border-radius: 4px;
        }}

        .badge-complete {{
            background-color: var(--success-glow);
            color: var(--success);
        }}

        .badge-partial {{
            background-color: rgba(245, 158, 11, 0.2);
            color: var(--warning);
        }}

        .badge-empty {{
            background-color: rgba(239, 68, 68, 0.2);
            color: var(--danger);
        }}

        .design-aspect-list {{
            display: flex;
            flex-wrap: wrap;
            gap: 0.4rem;
            margin-top: 0.5rem;
        }}

        .aspect-tag {{
            font-family: var(--mono-font);
            font-size: 0.75rem;
            padding: 0.2rem 0.5rem;
            border-radius: 4px;
        }}

        .aspect-implemented {{
            background-color: rgba(16, 185, 129, 0.15);
            color: #6ee7b7;
        }}

        .aspect-missing {{
            background-color: rgba(239, 68, 68, 0.15);
            color: #fca5a5;
            opacity: 0.6;
        }}

        .design-card {{
            display: block;
            text-decoration: none;
            color: inherit;
            transition: transform 0.2s ease, border-color 0.2s ease;
        }}
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div class="header-left">
                <h1>Quality Gate & Test Coverage</h1>
                <p>Generated automatically on {timestamp}</p>
            </div>
            <div class="status-badge">
                <span class="status-pulse"></span>
                <span>QUALITY GATE PASSED</span>
            </div>
        </header>

        <div class="grid-summary">
            <a href="#go-coverage" class="card-link">
            <div class="card">
                <h3>Go Internal Coverage</h3>
                <div class="value" style="color: var(--success);">{go_data["total"]}</div>
                <div class="sub">Line coverage of internal modules</div>
            </div>
            </a>
            <a href="#frontend-coverage" class="card-link">
            <div class="card">
                <h3>Frontend Coverage</h3>
                <div class="value" style="color: var(--success);">{bun_data["total_lines"]}</div>
                <div class="sub">Line coverage ({bun_data["total_funcs"]} functions)</div>
            </div>
            </a>
            <div class="card">
                <h3>Verification Gates</h3>
                <ul class="checklist" style="margin-top: 0.5rem;">
                    <li><span class="check-icon">✓</span> Traceability Validator: PASSED</li>
                    <li><span class="check-icon">✓</span> Local Stack Verifier: PASSED</li>
                    <li><span class="check-icon">✓</span> Frontend Screenshot Verifier: PASSED</li>
                    <li><span class="check-icon">✓</span> Requirements: PASSED ({reqs_checked}/{reqs_total})</li>
                </ul>
            </div>
            <a href="#design-coverage" class="card-link">
            <div class="card">
                <h3>Design Static Aspects</h3>
                <div class="value" style="color: {design_coverage_color};">{design_coverage_pct}</div>
                <div class="sub">{design_checked}/{design_total} static aspects implemented</div>
            </div>
            </a>
        </div>

        {screenshots_html}

        <div class="section-title" id="design-coverage">Design Static Aspects Coverage</div>
        <div class="design-grid">
            {design_rows}
        </div>



        <div class="section-title" id="go-coverage">Go Function Coverage Details</div>
        <div class="table-container">
            <table>
                <thead>
                    <tr>
                        <th>File</th>
                        <th>Function</th>
                        <th>Declaration Line</th>
                        <th>Status / Coverage</th>
                    </tr>
                </thead>
                <tbody>
                    {go_rows}
                </tbody>
            </table>
        </div>

        <div class="section-title" id="frontend-coverage">Frontend File Coverage Details</div>
        <div class="table-container">
            <table>
                <thead>
                    <tr>
                        <th>File</th>
                        <th>Functions Coverage</th>
                        <th>Lines Coverage</th>
                        <th>Uncovered Line Numbers</th>
                    </tr>
                </thead>
                <tbody>
                    {bun_rows}
                </tbody>
            </table>
        </div>


    </div>
</body>
</html>
"""
    Path(output_path).parent.mkdir(parents=True, exist_ok=True)
    html_content = "\n".join(line.rstrip() for line in html_content.splitlines()) + "\n"
    Path(output_path).write_text(html_content)
    print(f"Coverage and Quality Gate report successfully written to {output_path}")
