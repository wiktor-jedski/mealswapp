import { lookup } from "node:dns/promises";
import { BlockList, isIP } from "node:net";

// Implements DESIGN-014 LogAggregator deployed acceptance isolation.
const nonPublicIPv4 = new BlockList();
for (const [network, prefix] of [
	["0.0.0.0", 8],
	["10.0.0.0", 8],
	["100.64.0.0", 10],
	["127.0.0.0", 8],
	["169.254.0.0", 16],
	["172.16.0.0", 12],
	["192.0.0.0", 24],
	["192.0.2.0", 24],
	["192.88.99.0", 24],
	["192.168.0.0", 16],
	["198.18.0.0", 15],
	["198.51.100.0", 24],
	["203.0.113.0", 24],
	["224.0.0.0", 4],
	["240.0.0.0", 4]
] as const) {
	nonPublicIPv4.addSubnet(network, prefix, "ipv4");
}
const nonPublicIPv6 = new BlockList();
const publicIPv6 = new BlockList();
publicIPv6.addSubnet("2000::", 3, "ipv6");
for (const [network, prefix] of [
	["::", 128],
	["::1", 128],
	["::ffff:0:0", 96],
	["64:ff9b:1::", 48],
	["100::", 64],
	["2001::", 23],
	["2001:2::", 48],
	["2001:db8::", 32],
	["3fff::", 20],
	["5f00::", 16],
	["fc00::", 7],
	["fe80::", 10],
	["fec0::", 10],
	["ff00::", 8]
] as const) {
	nonPublicIPv6.addSubnet(network, prefix, "ipv6");
}

interface ResolvedAddress {
	address: string;
	family: number;
}

type Resolver = (hostname: string) => Promise<readonly ResolvedAddress[]>;

const resolveAll: Resolver = (hostname) => lookup(hostname, { all: true, verbatim: true });

function isPublicAddress({ address, family }: ResolvedAddress): boolean {
	const detectedFamily = isIP(address);
	return (
		(detectedFamily === 4 || detectedFamily === 6) &&
		detectedFamily === family &&
		(detectedFamily === 4 || publicIPv6.check(address, "ipv6")) &&
		!(detectedFamily === 4 ? nonPublicIPv4.check(address, "ipv4") : nonPublicIPv6.check(address, "ipv6"))
	);
}

/**
 * Validates and resolves the exact operator-approved public deployment origin.
 */
export async function validateTask285Target(
	baseURL: string | undefined,
	acknowledgement: string | undefined,
	approvedHostname: string | undefined,
	resolver: Resolver = resolveAll
): Promise<string> {
	const approvedHost = approvedHostname?.replace(/\.$/, "").toLowerCase();
	if (!baseURL || !approvedHost || acknowledgement !== "deployed-test-with-centralized-log-sink") {
		throw new Error("Task 285 requires an explicitly acknowledged deployed test environment.");
	}
	const target = new URL(baseURL);
	const hostname = target.hostname.replace(/^\[|\]$/g, "").replace(/\.$/, "").toLowerCase();
	if (
		target.protocol !== "https:" ||
		hostname !== approvedHost ||
		hostname === "localhost" ||
		isIP(hostname) !== 0 ||
		target.username ||
		target.password ||
		!["", "/"].includes(target.pathname) ||
		target.search ||
		target.hash
	) {
		throw new Error("Task 285 requires a credential-free deployed HTTPS origin.");
	}
	let addresses: readonly ResolvedAddress[];
	try {
		addresses = await resolver(hostname);
	} catch {
		throw new Error("Task 285 requires a publicly resolvable deployed HTTPS origin.");
	}
	if (!addresses.length || addresses.some((address) => !isPublicAddress(address))) {
		throw new Error("Task 285 requires a publicly resolvable deployed HTTPS origin.");
	}
	return baseURL;
}
