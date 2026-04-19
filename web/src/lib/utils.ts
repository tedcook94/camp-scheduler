import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}


export type WithElementRef<T, El extends HTMLElement = HTMLElement> = T & {
	ref?: El | null;
};

export type WithoutChild<T> = Omit<T, "child">;

export type WithoutChildrenOrChild<T> = Omit<T, "children" | "child">;

export type SortDirection = "asc" | "desc";

export type SortAccessor<T> = keyof T | ((item: T) => string | null | undefined);

export function sortItems<T>(
	items: T[],
	key: SortAccessor<T>,
	direction: SortDirection,
): T[] {
	const getValue = typeof key === "function" ? key : (item: T) => item[key] as unknown as string | null | undefined;

	return [...items].sort((a, b) => {
		const aVal = getValue(a);
		const bVal = getValue(b);

		// Nulls always sort last regardless of direction
		if (aVal == null && bVal == null) return 0;
		if (aVal == null) return 1;
		if (bVal == null) return -1;

		const cmp = String(aVal).localeCompare(String(bVal), undefined, { sensitivity: "base" });

		return direction === "asc" ? cmp : -cmp;
	});
}

export type PositiveIntResult =
	| { ok: true; value: number }
	| { ok: false; error: string };

export function parsePositiveInt(value: string | number | null | undefined): PositiveIntResult {
	if (value === null || value === undefined || value === "") {
		return { ok: false, error: "Required." };
	}
	const n = typeof value === "number" ? value : Number(String(value).trim());
	if (!Number.isFinite(n) || !Number.isInteger(n) || n < 1) {
		return { ok: false, error: "Must be a whole number 1 or greater." };
	}
	return { ok: true, value: n };
}
