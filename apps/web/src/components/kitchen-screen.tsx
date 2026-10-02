"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, ArrowRight, Check, ChefHat, Clock3, RefreshCw } from "lucide-react";
import { motion } from "framer-motion";
import { useEffect, useState } from "react";

import { AppShell } from "@/components/app-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ApiError, getKitchenOrders, queryKeys, type Order, type OrderStatus, updateOrderStatus } from "@/lib/api";
import { formatElapsed, formatPaise } from "@/lib/format";
import { useLiveEvents } from "@/hooks/use-live-events";

const columns: { status: OrderStatus; title: string; description: string; color: "blue" | "amber" | "green" | "gray" }[] = [
  { status: "new", title: "New", description: "Fresh tickets", color: "blue" },
  { status: "cooking", title: "Cooking", description: "In progress", color: "amber" },
  { status: "ready", title: "Ready", description: "Waiting to collect", color: "green" },
  { status: "picked_up", title: "Picked up", description: "Completed", color: "gray" },
];

const nextStatus: Partial<Record<OrderStatus, OrderStatus>> = { new: "cooking", cooking: "ready", ready: "picked_up" };

function useClock() {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const interval = window.setInterval(() => setNow(Date.now()), 15_000);
    return () => window.clearInterval(interval);
  }, []);
  return now;
}

function actionLabel(status: OrderStatus) {
  if (status === "new") return "Start cooking";
  if (status === "cooking") return "Mark ready";
  return "Mark picked up";
}

function OrderCard({ order, now, onAdvance, isUpdating }: { order: Order; now: number; onAdvance: (order: Order) => void; isUpdating: boolean }) {
  const next = nextStatus[order.status];
  const waitingMinutes = Math.floor((now - new Date(order.createdAt).getTime()) / 60_000);
  const needsAttention = order.status !== "picked_up" && waitingMinutes >= 10;
  const total = order.items.reduce((sum, item) => sum + item.unitPricePaise * item.quantity, 0);

  return (
    <motion.article layout initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{ type: "spring", stiffness: 340, damping: 28 }} className="rounded-2xl border border-black/[0.07] bg-white p-4 shadow-[0_8px_20px_rgb(30_34_30/0.04)] transition-shadow duration-300 hover:shadow-[0_14px_28px_rgb(30_34_30/0.09)]">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.14em] text-black/40">Order #{order.id}</p>
          <p className="mt-1 flex items-center gap-1.5 text-sm text-[#657064]"><Clock3 size={14} /> {formatElapsed(order.createdAt, now)}</p>
        </div>
        {needsAttention ? <Badge variant="solid" color="red" size="compact">Needs attention</Badge> : <Badge variant="dot" color="gray" size="compact">{formatPaise(total)}</Badge>}
      </div>
      <div className="my-4 h-px bg-black/[0.06]" />
      <ul className="space-y-2 text-sm">
        {order.items.map((item) => <li key={item.menuItemId} className="flex justify-between gap-3"><span><span className="font-semibold">{item.quantity}×</span> {item.menuItemName}</span><span className="text-black/45">{formatPaise(item.unitPricePaise * item.quantity)}</span></li>)}
      </ul>
      {next ? <Button className="mt-5 w-full" variant={order.status === "ready" ? "secondary" : "primary"} loading={isUpdating} onClick={() => onAdvance(order)} trailingIcon={ArrowRight}>{actionLabel(order.status)}</Button> : <div className="mt-5 flex items-center justify-center gap-2 rounded-xl bg-[#edf2de] py-2 text-sm font-medium text-[#526348]"><Check size={15} /> Complete</div>}
    </motion.article>
  );
}

export function KitchenScreen() {
  const queryClient = useQueryClient();
  const connectionState = useLiveEvents();
  const now = useClock();
  const ordersQuery = useQuery({ queryKey: queryKeys.kitchenOrders, queryFn: getKitchenOrders });
  const [activeOrderID, setActiveOrderID] = useState<number | null>(null);
  const updateMutation = useMutation({
    mutationFn: ({ order, status }: { order: Order; status: OrderStatus }) => updateOrderStatus(order.id, status),
    onSuccess: (updatedOrder) => {
      queryClient.setQueryData<Order[]>(queryKeys.kitchenOrders, (orders) => orders?.map((order) => order.id === updatedOrder.id ? updatedOrder : order));
      queryClient.setQueryData(queryKeys.order(updatedOrder.id), updatedOrder);
    },
    onSettled: () => {
      setActiveOrderID(null);
      queryClient.invalidateQueries({ queryKey: queryKeys.kitchenOrders });
    },
  });

  function advance(order: Order) {
    const status = nextStatus[order.status];
    if (!status) return;
    setActiveOrderID(order.id);
    updateMutation.mutate({ order, status });
  }

  const orders = ordersQuery.data ?? [];
  const openOrders = orders.filter((order) => order.status !== "picked_up").length;

  return (
    <AppShell connectionState={connectionState}>
      <main className="mx-auto max-w-[1600px] px-4 pb-12 pt-8 md:px-8 md:pt-12">
        <section className="mb-8 flex flex-wrap items-end justify-between gap-5">
          <div><p className="mb-2 text-sm font-medium uppercase tracking-[0.16em] text-[#778278]">Kitchen channel</p><h1 className="text-4xl font-semibold tracking-[-0.055em] sm:text-5xl">Keep the line moving.</h1><p className="mt-3 text-[#5f685f]">Every board receives the same live tickets and status changes.</p></div>
          <div className="flex w-full items-center gap-3 rounded-2xl border border-black/[0.07] bg-white px-4 py-3 shadow-sm sm:w-auto"><div className="grid size-9 place-items-center rounded-xl bg-[#edf2de] text-[#526348]"><ChefHat size={18} /></div><div><p className="text-xs uppercase tracking-[0.12em] text-black/45">Open orders</p><p className="text-lg font-semibold">{openOrders}</p></div></div>
        </section>

        {ordersQuery.isError ? <div className="rounded-3xl border border-red-200 bg-red-50 p-8 text-center"><AlertTriangle className="mx-auto text-red-600" /><p className="mt-3 font-semibold text-red-950">The board couldn&apos;t load.</p><p className="mt-1 text-sm text-red-800">Check the API connection, then try again.</p><Button variant="tertiary" className="mt-4" leadingIcon={RefreshCw} onClick={() => ordersQuery.refetch()}>Reload board</Button></div> : (
          <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-4">
            {columns.map((column) => {
              const columnOrders = orders.filter((order) => order.status === column.status);
              return <section key={column.status} className="min-h-80 rounded-3xl bg-[#e9e8e2] p-3"><div className="mb-4 flex items-center justify-between px-2 pt-2"><div><h2 className="font-semibold tracking-[-0.02em]">{column.title}</h2><p className="mt-0.5 text-xs text-black/45">{column.description}</p></div><Badge variant="solid" color={column.color}>{columnOrders.length}</Badge></div><div className="space-y-3">{ordersQuery.isLoading ? Array.from({ length: 2 }, (_, index) => <div key={index} className="h-44 animate-pulse rounded-2xl bg-white/70" />) : columnOrders.length === 0 ? <div className="rounded-2xl border border-dashed border-black/10 p-5 text-center text-sm text-black/40">No orders here</div> : columnOrders.map((order) => <OrderCard key={order.id} order={order} now={now} onAdvance={advance} isUpdating={activeOrderID === order.id && updateMutation.isPending} />)}</div></section>;
            })}
          </div>
        )}
        {updateMutation.isError ? <div className="fixed bottom-5 right-5 z-40 max-w-sm rounded-2xl bg-red-700 px-4 py-3 text-sm text-white shadow-xl">{updateMutation.error instanceof ApiError ? updateMutation.error.message : "Could not update the order. Please try again."}</div> : null}
      </main>
    </AppShell>
  );
}
