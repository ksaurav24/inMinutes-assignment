"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { apiURL, queryKeys, type MenuItem, type Order } from "@/lib/api";

type ConnectionState = "connecting" | "connected" | "reconnecting";

function mergeOrder(orders: Order[] | undefined, nextOrder: Order) {
	if (!orders) return [nextOrder];
	const existingIndex = orders.findIndex((order) => order.id === nextOrder.id);
	if (existingIndex === -1) return [...orders, nextOrder].sort((a, b) => a.createdAt.localeCompare(b.createdAt));
	return orders.map((order) => (order.id === nextOrder.id ? nextOrder : order));
}

export function useLiveEvents() {
	const queryClient = useQueryClient();
	const [connectionState, setConnectionState] = useState<ConnectionState>("connecting");

	useEffect(() => {
		const events = new EventSource(apiURL("/events"));
		events.onopen = () => setConnectionState("connected");
		events.onerror = () => setConnectionState("reconnecting");

		const updateMenu = (event: MessageEvent<string>) => {
			const menuItem = JSON.parse(event.data) as MenuItem;
			queryClient.setQueryData<MenuItem[]>(queryKeys.menu, (items) =>
				items?.map((item) => (item.id === menuItem.id ? menuItem : item))
			);
		};
		const updateOrder = (event: MessageEvent<string>) => {
			const order = JSON.parse(event.data) as Order;
			queryClient.setQueryData<Order>(queryKeys.order(order.id), order);
			queryClient.setQueryData<Order[]>(queryKeys.kitchenOrders, (orders) => mergeOrder(orders, order));
		};

		events.addEventListener("menu.updated", updateMenu);
		events.addEventListener("order.created", updateOrder);
		events.addEventListener("order.updated", updateOrder);

		return () => {
			events.close();
		};
	}, [queryClient]);

	return connectionState;
}
