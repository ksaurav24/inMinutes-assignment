"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { apiURL, queryKeys } from "@/lib/api";

type ConnectionState = "connecting" | "connected" | "reconnecting";

export function useLiveEvents() {
	const queryClient = useQueryClient();
	const [connectionState, setConnectionState] = useState<ConnectionState>("connecting");

	useEffect(() => {
		const events = new EventSource(apiURL("/events"));
		events.onopen = () => {
			setConnectionState("connected");
			void queryClient.cancelQueries().then(() => queryClient.invalidateQueries());
		};
		events.onerror = () => setConnectionState("reconnecting");

		const updateMenu = () => {
			void queryClient.invalidateQueries({ queryKey: queryKeys.menu });
		};
		const updateOrder = (event: MessageEvent<string>) => {
			const order = JSON.parse(event.data) as { id: number };
			void queryClient.invalidateQueries({ queryKey: queryKeys.order(order.id) });
			void queryClient.invalidateQueries({ queryKey: queryKeys.kitchenOrders });
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
