import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

export function capitalizeFirst(s: string): string {
	return s.charAt(0).toUpperCase() + s.slice(1);
}

export type WithElementRef<T, El extends HTMLElement = HTMLElement> = T & {
	ref?: El | null;
};

export type WithoutChild<T> = Omit<T, "child">;

export type WithoutChildrenOrChild<T> = Omit<T, "children" | "child">;
