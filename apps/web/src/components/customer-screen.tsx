"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, Check, CreditCard, Minus, Plus, RefreshCw, ShoppingBag } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { useMemo, useState, useSyncExternalStore } from "react";

import { AppShell } from "@/components/app-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ApiError, createOrder, getMenu, getOrder, queryKeys, type MenuItem } from "@/lib/api";
import { formatPaise, statusLabel } from "@/lib/format";
import { useLiveEvents } from "@/hooks/use-live-events";

type StockIssue = {
	menuItemId: number;
	name: string;
	available: number;
	requested: number;
};

const emptyMenu: MenuItem[] = [];
const lastOrderStorageKey = "inminutes:last-order-id";
const lastOrderStorageEvent = "inminutes:last-order-changed";

function createIdempotencyKey() {
	return crypto.randomUUID();
}

function readLastOrderID() {
	const value = Number(window.localStorage.getItem(lastOrderStorageKey));
	return Number.isSafeInteger(value) && value > 0 ? value : null;
}

function subscribeToLastOrder(callback: () => void) {
	window.addEventListener("storage", callback);
	window.addEventListener(lastOrderStorageEvent, callback);
	return () => {
		window.removeEventListener("storage", callback);
		window.removeEventListener(lastOrderStorageEvent, callback);
	};
}

function saveLastOrderID(orderID: number) {
	window.localStorage.setItem(lastOrderStorageKey, String(orderID));
	window.dispatchEvent(new Event(lastOrderStorageEvent));
}

function clearLastOrderID() {
	window.localStorage.removeItem(lastOrderStorageKey);
	window.dispatchEvent(new Event(lastOrderStorageEvent));
}

function QueryFailure({ onRetry }: { onRetry: () => void }) {
	return (
		<div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-center">
			<AlertTriangle className="mx-auto mb-3 text-red-600" aria-hidden />
			<p className="font-semibold text-red-950">Couldn&apos;t load the menu</p>
			<p className="mt-1 text-sm text-red-800">The kitchen may be temporarily unavailable.</p>
			<Button variant="tertiary" className="mt-4" leadingIcon={RefreshCw} onClick={onRetry}>
				Try again
			</Button>
		</div>
	);
}

function StockMessage({ error }: { error: ApiError }) {
	const issues = unavailableItems(error);
	if (error.code !== "out_of_stock" || issues.length === 0) return <p>{error.message}</p>;

	return (
		<div>
			<p className="font-semibold">A few items changed while you were checking out.</p>
			<ul className="mt-2 space-y-1 text-sm">
				{issues.map((issue) => (
					<li key={issue.menuItemId}>
						{issue.name || "Menu item"}: {issue.available === 0 ? "now sold out" : `${issue.available} left`}
					</li>
				))}
			</ul>
		</div>
	);
}

function unavailableItems(error: unknown) {
	if (!(error instanceof ApiError) || error.code !== "out_of_stock") return [];
	return (error.details as { items?: StockIssue[] } | undefined)?.items ?? [];
}

export function CustomerScreen() {
	const queryClient = useQueryClient();
	const connectionState = useLiveEvents();
	const menuQuery = useQuery({ queryKey: queryKeys.menu, queryFn: getMenu });
	const [cart, setCart] = useState<Record<number, number>>({});
	const [checkoutKey, setCheckoutKey] = useState<string | null>(null);
	const lastOrderID = useSyncExternalStore(subscribeToLastOrder, readLastOrderID, () => null);

	const orderQuery = useQuery({
		queryKey: queryKeys.order(lastOrderID ?? 0),
		queryFn: () => getOrder(lastOrderID!),
		enabled: lastOrderID !== null,
	});

	const orderMutation = useMutation({
		mutationFn: ({ items, idempotencyKey }: { items: { menuItemId: number; quantity: number }[]; idempotencyKey: string }) =>
			createOrder(items, idempotencyKey),
		onSuccess: (order) => {
			queryClient.setQueryData(queryKeys.order(order.id), order);
			queryClient.invalidateQueries({ queryKey: queryKeys.menu });
			saveLastOrderID(order.id);
			setCart({});
			setCheckoutKey(null);
		},
	});

	const menuItems = menuQuery.data ?? emptyMenu;
	const menuByID = useMemo(() => new Map(menuItems.map((item) => [item.id, item])), [menuItems]);
	const cartItems = useMemo(
		() =>
			Object.entries(cart)
				.map(([id, quantity]) => ({ item: menuByID.get(Number(id)), quantity }))
				.filter((line): line is { item: MenuItem; quantity: number } => Boolean(line.item) && line.quantity > 0),
		[cart, menuByID]
	);
	const total = cartItems.reduce((sum, line) => sum + line.item.pricePaise * line.quantity, 0);
	const itemCount = cartItems.reduce((sum, line) => sum + line.quantity, 0);
	const latestOrder = orderQuery.data;
	const savedOrderMissing = orderQuery.error instanceof ApiError && orderQuery.error.code === "not_found";
	const unavailableCartItems = unavailableItems(orderMutation.error);

	function changeQuantity(item: MenuItem, delta: number) {
		setCart((current) => {
			const nextQuantity = Math.max(0, (current[item.id] ?? 0) + delta);
			const next = { ...current };
			if (nextQuantity === 0) delete next[item.id];
			else next[item.id] = nextQuantity;
			return next;
		});
	}

	function checkout() {
		if (cartItems.length === 0 || orderMutation.isPending) return;
		const idempotencyKey = checkoutKey ?? createIdempotencyKey();
		setCheckoutKey(idempotencyKey);
		orderMutation.mutate({
			idempotencyKey,
			items: cartItems.map(({ item, quantity }) => ({ menuItemId: item.id, quantity })),
		});
	}

	function clearSavedOrder() {
		clearLastOrderID();
	}

	function removeUnavailableItems() {
		if (unavailableCartItems.length === 0) return;
		setCart((current) => {
			const next = { ...current };
			for (const item of unavailableCartItems) delete next[item.menuItemId];
			return next;
		});
		setCheckoutKey(null);
		orderMutation.reset();
		queryClient.invalidateQueries({ queryKey: queryKeys.menu });
	}

	return (
		<AppShell connectionState={connectionState}>
			<main className="mx-auto max-w-7xl px-4 pb-28 pt-8 md:px-8 md:pt-12 lg:pb-24">
				<section className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_360px] lg:items-start">
					<div>
						<div className="mb-8 flex flex-wrap items-end justify-between gap-4">
							<div>
								<p className="mb-2 text-sm font-medium uppercase tracking-[0.16em] text-[#778278]">Order channel</p>
								<h1 className="text-4xl font-semibold tracking-[-0.055em] text-[#1d211e] sm:text-5xl">Lunch, on your terms.</h1>
								<p className="mt-3 max-w-xl text-base leading-7 text-[#5f685f]">Real stock, real-time updates, and a quick mock checkout.</p>
							</div>
							<Badge variant="solid" color="green">Kitchen is taking orders</Badge>
						</div>

						{menuQuery.isLoading ? (
							<div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
								{Array.from({ length: 6 }, (_, index) => <div key={index} className="h-52 animate-pulse rounded-3xl bg-[#e9e8e2]" />)}
							</div>
						) : menuQuery.isError ? (
							<QueryFailure onRetry={() => menuQuery.refetch()} />
						) : (
							<div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
								{menuItems.map((item) => {
									const quantity = cart[item.id] ?? 0;
									const soldOut = item.stock === 0;
									return (
									<motion.article
										key={item.id}
										initial={{ opacity: 0, y: 14 }}
										animate={{ opacity: 1, y: 0 }}
										transition={{ type: "spring", stiffness: 320, damping: 26, delay: Math.min(item.id * 0.035, 0.2) }}
										className="group min-h-56 rounded-3xl border border-black/[0.07] bg-white p-5 shadow-[0_12px_32px_rgb(40_45_40/0.04)] transition-[transform,box-shadow,border-color] duration-300 ease-out hover:-translate-y-1 hover:border-black/[0.12] hover:shadow-[0_18px_40px_rgb(40_45_40/0.1)]"
									>
											<div className="flex items-start justify-between gap-3">
												<div className="grid size-12 place-items-center rounded-2xl bg-[#edf2de] text-xl" aria-hidden>{item.name.split(" ")[0].slice(0, 1)}</div>
												<Badge variant="dot" color={soldOut ? "red" : item.stock <= 3 ? "amber" : "green"} size="compact">
													{soldOut ? "Sold out" : `${item.stock} left`}
												</Badge>
											</div>
											<div className="mt-7">
												<h2 className="text-lg font-semibold tracking-[-0.03em]">{item.name}</h2>
												<p className="mt-1 text-sm text-[#6c746c]">Made fresh when your ticket lands.</p>
											</div>
											<div className="mt-5 flex items-center justify-between">
												<span className="font-semibold">{formatPaise(item.pricePaise)}</span>
											<AnimatePresence initial={false} mode="popLayout">
												{quantity > 0 ? (
													<motion.div key="stepper" initial={{ opacity: 0, scale: 0.82 }} animate={{ opacity: 1, scale: 1 }} exit={{ opacity: 0, scale: 0.82 }} transition={{ type: "spring", stiffness: 420, damping: 25 }} className="flex items-center gap-2 rounded-xl bg-[#f3f3ef] p-1">
														<Button aria-label={`Remove one ${item.name}`} variant="ghost" size="icon-compact" disabled={orderMutation.isPending} onClick={() => changeQuantity(item, -1)}><Minus /></Button>
														<motion.span key={quantity} initial={{ opacity: 0, y: -4 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.16 }} className="min-w-4 text-center text-sm font-semibold">{quantity}</motion.span>
														<Button aria-label={`Add one ${item.name}`} variant="ghost" size="icon-compact" disabled={orderMutation.isPending || quantity >= item.stock} onClick={() => changeQuantity(item, 1)}><Plus /></Button>
													</motion.div>
												) : (
													<motion.div key="add" initial={{ opacity: 0, scale: 0.82 }} animate={{ opacity: 1, scale: 1 }} exit={{ opacity: 0, scale: 0.82 }} transition={{ type: "spring", stiffness: 420, damping: 25 }}><Button variant="secondary" size="compact" disabled={soldOut || orderMutation.isPending} leadingIcon={Plus} onClick={() => changeQuantity(item, 1)}>Add</Button></motion.div>
												)}
											</AnimatePresence>
										</div>
									</motion.article>
									);
								})}
							</div>
						)}
					</div>

					<aside className="rounded-3xl bg-[#1d211e] p-5 text-[#f7f6f0] shadow-[0_24px_60px_rgb(29_33_30/0.2)] lg:sticky lg:top-24">
						<div className="flex items-center justify-between">
							<div>
								<p className="text-sm text-[#b9c3b8]">Your cart</p>
								<p className="mt-1 text-xl font-semibold">{itemCount === 0 ? "Nothing yet" : `${itemCount} item${itemCount === 1 ? "" : "s"}`}</p>
							</div>
							<div className="grid size-10 place-items-center rounded-2xl bg-[#303631] text-[#f9e96c]"><ShoppingBag size={18} /></div>
						</div>

						<div className="my-5 h-px bg-white/10" />
						{cartItems.length === 0 ? (
							<p className="py-8 text-center text-sm leading-6 text-[#b9c3b8]">Choose something from the menu. Your cart updates against live stock.</p>
						) : (
							<AnimatePresence initial={false} mode="popLayout">
								<div className="space-y-4">
									{cartItems.map(({ item, quantity }) => <motion.div key={item.id} layout initial={{ opacity: 0, x: 12 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: 12 }} transition={{ type: "spring", stiffness: 360, damping: 30 }} className="flex justify-between gap-3 text-sm"><span><span className="font-medium">{quantity}×</span> <span className="text-[#dbe2d9]">{item.name}</span></span><span>{formatPaise(item.pricePaise * quantity)}</span></motion.div>)}
								</div>
							</AnimatePresence>
						)}

						<div className="mt-6 rounded-2xl bg-white/[0.08] p-4">
							<div className="flex items-center gap-3"><CreditCard className="text-[#f9e96c]" size={18} /><div><p className="text-sm font-medium">Mock payment</p><p className="text-xs text-[#b9c3b8]">Visa ending in 4242 · no charge is made</p></div></div>
							<div className="mt-4 flex items-end justify-between"><span className="text-sm text-[#b9c3b8]">Total</span><span className="text-2xl font-semibold tracking-[-0.04em]">{formatPaise(total)}</span></div>
						</div>

						{orderMutation.isError && <div className="mt-4 rounded-2xl bg-red-500/15 p-4 text-sm text-red-100"><StockMessage error={orderMutation.error as ApiError} /></div>}
						{unavailableCartItems.length > 0 ? <Button variant="secondary" className="mt-3 w-full bg-[#f9e96c] text-[#1d211e] hover:bg-[#fff29e]" leadingIcon={Minus} onClick={removeUnavailableItems}>Remove unavailable items</Button> : null}
						{orderMutation.isError && unavailableCartItems.length === 0 ? <Button variant="ghost" className="mt-3 w-full text-[#f9e96c] hover:text-[#f9e96c]" onClick={() => { orderMutation.reset(); queryClient.invalidateQueries({ queryKey: queryKeys.menu }); }}>Refresh live stock</Button> : null}
						<Button className="mt-5 hidden w-full bg-[#f9e96c] text-[#1d211e] hover:bg-[#fff29e] lg:inline-flex" variant="secondary" loading={orderMutation.isPending} disabled={cartItems.length === 0} onClick={checkout} leadingIcon={CreditCard}>
							{orderMutation.isPending ? "Authorizing mock payment" : `Pay ${formatPaise(total)}`}
						</Button>
					</aside>
				</section>

				<AnimatePresence>
					{cartItems.length > 0 ? <motion.div initial={{ opacity: 0, y: 24 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: 24 }} transition={{ type: "spring", stiffness: 380, damping: 30 }} className="fixed inset-x-0 bottom-0 z-40 border-t border-black/[0.08] bg-[#f6f5f0]/95 px-4 py-3 shadow-[0_-12px_28px_rgb(29_33_30/0.1)] backdrop-blur lg:hidden"><div className="mx-auto flex max-w-xl items-center gap-3"><div className="min-w-0 flex-1"><p className="text-xs text-[#657064]">{itemCount} item{itemCount === 1 ? "" : "s"} · mock payment</p><p className="truncate text-lg font-semibold">{formatPaise(total)}</p></div><Button variant="secondary" className="shrink-0 bg-[#f9e96c] text-[#1d211e] hover:bg-[#fff29e]" loading={orderMutation.isPending} onClick={checkout} leadingIcon={CreditCard}>{orderMutation.isPending ? "Paying" : `Pay ${formatPaise(total)}`}</Button></div></motion.div> : null}
				</AnimatePresence>

				{latestOrder && <section className="mt-8 rounded-3xl border border-emerald-200 bg-emerald-50 p-6 shadow-sm"><div className="flex flex-wrap items-start justify-between gap-4"><div className="flex gap-4"><div className="grid size-10 place-items-center rounded-2xl bg-emerald-600 text-white"><Check size={20} /></div><div><p className="font-semibold text-emerald-950">Order #{latestOrder.id} is with the kitchen</p><p className="mt-1 text-sm text-emerald-800">Saved on this device and kept in sync with the kitchen.</p></div></div><Badge variant="solid" color="green">{statusLabel(latestOrder.status)}</Badge></div></section>}
				{savedOrderMissing && <section className="mt-8 rounded-3xl border border-amber-200 bg-amber-50 p-6"><p className="font-semibold text-amber-950">Your saved order is no longer available.</p><p className="mt-1 text-sm text-amber-800">It may have been removed when the local database was reset.</p><Button variant="tertiary" className="mt-4" onClick={clearSavedOrder}>Clear saved order</Button></section>}
			</main>
		</AppShell>
	);
}
