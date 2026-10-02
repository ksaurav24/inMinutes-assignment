export type MenuItem = {
  id: number;
  name: string;
  pricePaise: number;
  stock: number;
};

export type OrderStatus = "new" | "cooking" | "ready" | "picked_up";

export type OrderItem = {
  menuItemId: number;
  menuItemName: string;
  quantity: number;
  unitPricePaise: number;
};

export type Order = {
  id: number;
  status: OrderStatus;
  items: OrderItem[];
  createdAt: string;
  updatedAt: string;
};

type ApiEnvelope<T> = { data: T };

type ApiFailure = {
  error?: {
    code?: string;
    message?: string;
    details?: unknown;
  };
};

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code: string,
    readonly details?: unknown
  ) {
    super(message);
  }
}

const apiBaseURL = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/\/$/, "");

export const queryKeys = {
  menu: ["menu"] as const,
  kitchenOrders: ["kitchen-orders"] as const,
  order: (id: number) => ["order", id] as const,
};

export function apiURL(path: string) {
  return `${apiBaseURL}${path}`;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	let response: Response;
	try {
		response = await fetch(apiURL(path), {
			...init,
			headers: {
				Accept: "application/json",
				...init?.headers,
			},
		});
	} catch {
		throw new ApiError("Could not reach the kitchen. Check your connection and try again.", 0, "network_error");
	}

	const payload = (await response.json().catch(() => null)) as ApiEnvelope<T> | ApiFailure | null;
	if (!response.ok) {
		const failure = payload as ApiFailure | null;
		throw new ApiError(
			failure?.error?.message ?? "Something went wrong. Please try again.",
			response.status,
			failure?.error?.code ?? "request_failed",
			failure?.error?.details
		);
	}
	if (!payload || !("data" in payload)) {
		throw new ApiError("The kitchen sent an unexpected response.", response.status, "invalid_response");
	}
	return payload.data;
}

export function getMenu() {
	return request<{ items: MenuItem[] }>("/menu").then((response) => response.items);
}

export function getOrder(id: number) {
	return request<Order>(`/orders/${id}`);
}

export function getKitchenOrders() {
	return request<{ orders: Order[] }>("/kitchen/orders").then((response) => response.orders);
}

export function createOrder(items: { menuItemId: number; quantity: number }[], idempotencyKey: string) {
	return request<Order>("/orders", {
		method: "POST",
		headers: {
			"Content-Type": "application/json",
			"Idempotency-Key": idempotencyKey,
		},
		body: JSON.stringify({ items }),
	});
}

export function updateOrderStatus(id: number, status: OrderStatus) {
	return request<Order>(`/orders/${id}`, {
		method: "PATCH",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ status }),
	});
}
