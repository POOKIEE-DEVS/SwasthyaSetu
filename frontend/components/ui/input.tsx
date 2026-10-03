import * as React from "react";

import { cn } from "@/lib/utils";

function Input({ className, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      data-slot="input"
      className={cn(
        "flex h-11 w-full rounded-[8px] border border-input bg-card px-3 py-2 text-base",
        "placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:border-ring",
        "focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
      {...props}
    />
  );
}

export { Input };
