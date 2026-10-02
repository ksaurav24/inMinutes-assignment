import type { OrderStatus } from "@/lib/api";

export function formatPaise(value: number) {
	return new Intl.NumberFormat("en-IN", {
		style: "currency",
		currency: "INR",
		maximumFractionDigits: 2,
	}).format(value / 100);
}

export function formatElapsed(isoDate: string, now = Date.now()) {
	const elapsedSeconds = Math.max(0, Math.floor((now - new Date(isoDate).getTime()) / 1000));
	if (elapsedSeconds < 60) return "just now";
	if (elapsedSeconds < 3600) return `${Math.floor(elapsedSeconds / 60)}m ago`;
	return `${Math.floor(elapsedSeconds / 3600)}h ago`;
}

export function statusLabel(status: OrderStatus) {
	return {
		new: "New",
		cooking: "Cooking",
		ready: "Ready",
		picked_up: "Picked up",
	}[status];
}
