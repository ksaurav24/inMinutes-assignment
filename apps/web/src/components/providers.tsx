"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState, type ReactNode } from "react";

import { ShapeProvider } from "@/lib/shape-context";
import { SizeProvider } from "@/lib/size-context";

export function Providers({ children }: { children: ReactNode }) {
	const [queryClient] = useState(
		() =>
			new QueryClient({
				defaultOptions: {
					queries: { retry: 1, refetchOnWindowFocus: false },
				},
			})
	);

	return (
		<QueryClientProvider client={queryClient}>
			<ShapeProvider>
				<SizeProvider>{children}</SizeProvider>
			</ShapeProvider>
		</QueryClientProvider>
	);
}
