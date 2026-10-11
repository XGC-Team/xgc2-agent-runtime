// T3 Code, MIT, Copyright (c) 2026 T3 Tools Inc.
// Upstream ref: 477282263a9115f911380b5aec7d4973267e4edc. See UPSTREAM.md.
import { Separator as SeparatorPrimitive } from "@base-ui/react/separator";

import { cn } from "../utils.js";

function Separator({ className, orientation = "horizontal", ...props }: SeparatorPrimitive.Props) {
  return (
    <SeparatorPrimitive
      className={cn(
        "shrink-0 bg-border data-[orientation=horizontal]:h-px data-[orientation=horizontal]:w-full data-[orientation=vertical]:w-px data-[orientation=vertical]:not-[[class^='h-']]:not-[[class*='_h-']]:self-stretch",
        className,
      )}
      data-slot="separator"
      orientation={orientation}
      {...props}
    />
  );
}

export { Separator };
