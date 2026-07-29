import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import type {
	FullConfig,
	FullResult,
	Reporter,
	Suite,
	TestCase,
	TestResult
} from "@playwright/test/reporter";

// Implements DESIGN-014 MetricsCollector and DESIGN-009 AdminController acceptance evidence.

interface AcceptanceAttachment {
	criterionIds: string[];
	rootCauseId?: string;
	requestIds: string[];
	evidence: Array<{ type: "playwright" | "backend"; path: string }>;
	backendEvidence: string[];
}

interface CriterionRun {
	status: TestResult["status"];
	project: string;
	attachment?: AcceptanceAttachment;
}

const CRITERION_PATTERN = /P08-SWR\d{3}-(?:STEP|ACCEPT)-\d{2}/g;
const CRITERION_ID_PATTERN = /^P08-SWR\d{3}-(?:STEP|ACCEPT)-\d{2}$/;
const BACKEND_EVIDENCE_PATTERN = /^[a-z][a-z0-9_]*=[a-zA-Z0-9._:-]+$/;
const BACKEND_EVIDENCE_KEYS = new Set(["http_status", "mutation_count", "audit_count", "row_count", "owner_state", "cache_generation", "worker_state", "provider_state", "log_sink_state", "metric_basis", "export_record_count", "request_correlation", "rollback_state"]);
const DEFAULT_INFRASTRUCTURE_ROOT = "ROOT-T281-ACCEPTANCE-INFRASTRUCTURE";

/** Collects isolated Playwright outcomes into the Task 280 producer contract. */
export default class Phase08AcceptanceReporter implements Reporter {
	private readonly runs = new Map<string, CriterionRun[]>();
	private readonly projects = new Set<string>();

	onBegin(_config: FullConfig, _suite: Suite): void {
		this.runs.clear();
		this.projects.clear();
	}

	onTestEnd(test: TestCase, result: TestResult): void {
		const project = test.parent.project()?.name ?? "unknown";
		this.projects.add(project);
		const titleCriteria = [...new Set(test.title.match(CRITERION_PATTERN) ?? [])];
		const attachments = result.attachments
			.filter((item) => item.name === "phase08-acceptance" && item.body)
			.map((item) => parseAttachment(item.body!.toString("utf8")))
			.filter((item): item is AcceptanceAttachment => item !== undefined);
		const attachment = mergeAttachments(attachments);
		for (const criterionId of new Set([
			...titleCriteria,
			...(attachment?.criterionIds ?? [])
		])) {
			const runs = this.runs.get(criterionId) ?? [];
			runs.push({ status: result.status, project: test.parent.project()?.name ?? "unknown", attachment });
			this.runs.set(criterionId, runs);
		}
	}

	onEnd(_result: FullResult): void {
		const outputDirectory = process.env.PHASE08_ACCEPTANCE_RESULT_DIR;
		const resultName = process.env.MEALSWAPP_PHASE08_RESULT_FILE;
		const expectedCriteria = splitEnvironment("MEALSWAPP_PHASE08_CRITERIA");
		const expectedProjects = new Set(splitEnvironment("MEALSWAPP_PHASE08_EXPECTED_PROJECTS"));
		const infrastructureRoot = process.env.MEALSWAPP_PHASE08_INFRASTRUCTURE_ROOT ?? DEFAULT_INFRASTRUCTURE_ROOT;
		const synchronizedRoots = new Set([
			infrastructureRoot,
			...splitEnvironment("MEALSWAPP_PHASE08_SYNCHRONIZED_ROOTS")
		]);
		if (!outputDirectory || !resultName || expectedCriteria.length === 0 || expectedProjects.size === 0) {
			throw new Error("Phase 08 reporter environment is incomplete");
		}
		const results = expectedCriteria.map((criterionId) => {
			const runs = this.runs.get(criterionId) ?? [];
			const projects = new Set(runs.map((run) => run.project));
			const statuses = new Set(runs.map((run) => run.status));
			const status =
				statuses.has("failed") || statuses.has("timedOut") || statuses.has("interrupted")
					? "FAIL"
					: runs.length === 0 ||
						  statuses.has("skipped") ||
						  [...expectedProjects].some((project) => !projects.has(project))
						? "BLOCKED"
						: "PASS";
			const attachments = runs.flatMap((run) => run.attachment ? [run.attachment] : []);
			return {
				criterionId,
				status,
				...(status === "PASS" ? {} : {
					rootCauseId:
						attachments.find((item) => synchronizedRoots.has(item.rootCauseId ?? ""))?.rootCauseId ??
						infrastructureRoot
				}),
				requestIds: [...new Set(attachments.flatMap((item) => item.requestIds))].sort(),
				evidence: uniqueEvidence(attachments.flatMap((item) => item.evidence)),
				backendEvidence: [...new Set(attachments.flatMap((item) => item.backendEvidence))].sort()
			};
		});
		mkdirSync(outputDirectory, { recursive: true });
		writeFileSync(join(outputDirectory, resultName), `${JSON.stringify({ results, projects: [...this.projects].sort() }, null, 2)}\n`, {
			encoding: "utf8",
			mode: 0o600
		});
	}
}

function parseAttachment(value: string): AcceptanceAttachment | undefined {
	try {
		const parsed: unknown = JSON.parse(value);
		if (
			!isRecord(parsed) ||
			!Array.isArray(parsed.criterionIds) ||
			!parsed.criterionIds.every((item) => typeof item === "string" && CRITERION_ID_PATTERN.test(item)) ||
			(parsed.rootCauseId !== undefined && typeof parsed.rootCauseId !== "string") ||
			!Array.isArray(parsed.requestIds) ||
			!parsed.requestIds.every((item) => typeof item === "string") ||
			!Array.isArray(parsed.evidence) ||
			!parsed.evidence.every(
				(item) =>
					isRecord(item) &&
					Object.keys(item).length === 2 &&
					(item.type === "playwright" || item.type === "backend") &&
					typeof item.path === "string"
			) ||
			!Array.isArray(parsed.backendEvidence) ||
			!parsed.backendEvidence.every((item) => typeof item === "string" && BACKEND_EVIDENCE_PATTERN.test(item) && BACKEND_EVIDENCE_KEYS.has(item.split("=", 1)[0]))
		) return undefined;
		return parsed as AcceptanceAttachment;
	} catch {
		return undefined;
	}
}

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null && !Array.isArray(value);
}

function splitEnvironment(name: string): string[] {
	return (process.env[name] ?? "").split(",").map((value) => value.trim()).filter(Boolean);
}

function uniqueEvidence(
	evidence: AcceptanceAttachment["evidence"]
): AcceptanceAttachment["evidence"] {
	return [...new Map(evidence.map((item) => [`${item.type}:${item.path}`, item])).values()]
		.sort((left, right) => `${left.type}:${left.path}`.localeCompare(`${right.type}:${right.path}`));
}

function mergeAttachments(attachments: AcceptanceAttachment[]): AcceptanceAttachment | undefined {
	if (attachments.length === 0) return undefined;
	return {
		criterionIds: [...new Set(attachments.flatMap((item) => item.criterionIds))],
		requestIds: [...new Set(attachments.flatMap((item) => item.requestIds))],
		evidence: uniqueEvidence(attachments.flatMap((item) => item.evidence)),
		backendEvidence: [...new Set(attachments.flatMap((item) => item.backendEvidence))],
		...([...attachments].reverse().find((item) => item.rootCauseId)?.rootCauseId
			? { rootCauseId: [...attachments].reverse().find((item) => item.rootCauseId)!.rootCauseId }
			: {})
	};
}
