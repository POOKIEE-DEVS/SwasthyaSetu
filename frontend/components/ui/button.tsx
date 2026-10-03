import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

// Controls use the 12px radius (major surfaces use 20px). Feedback is a
// colour change on hover and a 1px press on click; no gradients, no glow.
// On phones every button is at least 44px tall; compact again from `sm`.
const buttonVariants = cva(
  "inline-flex cursor-pointer items-center justify-center gap-2 whitespace-nowrap rounded-[12px] text-sm font-semibold transition-[background-color,color,border-color,transform] duration-150 ease-[var(--ease-out)] active:translate-y-px focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary/90",
        // Kept as an alias of the primary action for existing screens.
        cta: "bg-primary text-primary-foreground hover:bg-primary/90",
        destructive: "bg-destructive text-destructive-foreground hover:bg-destructive/90",
        outline:
          "border border-input bg-card text-foreground hover:border-primary/60 hover:bg-accent",
        secondary: "bg-secondary text-secondary-foreground hover:bg-accent",
        ghost: "text-foreground/80 hover:bg-accent hover:text-foreground",
        link: "rounded-none text-primary-text underline-offset-4 hover:underline",
        // Emergency only: calling 102. Its own variant so it can never look
        // like a delete button, and red is never used for anything else.
        emergency: "bg-triage-red text-triage-red-foreground hover:bg-triage-red/90",
      },
      size: {
        default: "h-11 px-5 sm:h-10",
        sm: "h-11 px-3.5 text-[13px] sm:h-9",
        lg: "h-12 px-6 text-[15px]",
        icon: "size-11 sm:size-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

function Button({
  className,
  variant,
  size,
  asChild = false,
  ...props
}: React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot : "button";
  return (
    <Comp
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  );
}

export { Button, buttonVariants };
