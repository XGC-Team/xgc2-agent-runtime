// T3 Code, MIT, Copyright (c) 2026 T3 Tools Inc.
// Upstream ref: 477282263a9115f911380b5aec7d4973267e4edc. See UPSTREAM.md for integration changes.
import { Loader2Icon } from "lucide-react";
import { cn } from "../utils.js";

function Spinner({ className, ...props }: React.ComponentProps<typeof Loader2Icon>) {
  return (
    <Loader2Icon
      aria-label="Loading"
      className={cn("animate-spin", className)}
      role="status"
      {...props}
    />
  );
}

export { Spinner };
