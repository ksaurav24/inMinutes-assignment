"use client";

import { ChefHat, ShoppingBag } from "lucide-react";
import { motion } from "framer-motion";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

type AppShellProps = {
	children: ReactNode;
	connectionState: "connecting" | "connected" | "reconnecting";
};

export function AppShell({ children, connectionState }: AppShellProps) {
	const pathname = usePathname();
	const activeChannel = pathname === "/kitchen" ? "kitchen" : "customer";

	return (
		<div className="min-h-screen overflow-x-hidden bg-[#f6f5f0] text-[#1d211e]">
			<header className="sticky top-0 z-30 border-b border-black/[0.07] bg-[#f6f5f0]/90 px-4 py-3 backdrop-blur md:px-8">
				<div className="mx-auto flex max-w-7xl flex-wrap items-center gap-3 sm:flex-nowrap sm:gap-4">
					<div className="flex min-w-0 flex-1 items-center gap-3">
						<div className="grid size-9 shrink-0 place-items-center rounded-xl bg-[#1d211e] text-[#f9e96c] shadow-sm">
							<ChefHat size={20} strokeWidth={2.2} />
						</div>
						<div>
							<p className="text-sm font-semibold tracking-[-0.02em]">inMinutes</p>
							<p className="hidden text-xs text-black/45 sm:block">Live kitchen operations</p>
						</div>
					</div>

					<nav aria-label="Choose a channel" className="order-3 flex w-full items-center gap-1 rounded-xl bg-black/[0.04] p-1 sm:order-2 sm:w-auto">
						<Button asChild className="flex-1 sm:flex-none" variant={activeChannel === "customer" ? "secondary" : "ghost"} size="compact" leadingIcon={ShoppingBag}>
							<Link href="/">Order</Link>
						</Button>
						<Button asChild className="flex-1 sm:flex-none" variant={activeChannel === "kitchen" ? "secondary" : "ghost"} size="compact" leadingIcon={ChefHat}>
							<Link href="/kitchen">Kitchen</Link>
						</Button>
					</nav>

					<Badge variant="dot" color={connectionState === "connected" ? "green" : "amber"} size="compact">
						<span className="hidden sm:inline">{connectionState === "connected" ? "Live sync" : "Reconnecting"}</span>
						<span className="sm:hidden">{connectionState === "connected" ? "Live" : "Syncing"}</span>
					</Badge>
				</div>
			</header>
			<motion.div
				key={pathname}
				initial={{ opacity: 0, y: 10 }}
				animate={{ opacity: 1, y: 0 }}
				transition={{ type: "spring", stiffness: 320, damping: 28, mass: 0.7 }}
			>
				{children}
			</motion.div>
		</div>
	);
}
