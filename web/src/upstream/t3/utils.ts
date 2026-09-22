// T3 Code, MIT, Copyright (c) 2026 T3 Tools Inc.
// Upstream ref: bf3be75c400cf605dc0c80de9458458854c84131. See UPSTREAM.md for integration changes.
import { type CxOptions, cx } from "class-variance-authority";
import { twMerge } from "tailwind-merge";
export function cn(...inputs: CxOptions) { return twMerge(cx(inputs)); }
export function isMacPlatform(platform: string): boolean { return /mac|iphone|ipad|ipod/i.test(platform); }
