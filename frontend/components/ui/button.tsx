import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

// Buttons are full pills with a soft raised shadow that presses in on
// click (soft UI). On phones every button is at least 44px tall, the
// minimum comfortable touch target; from `sm` up they return to compact.
const buttonVariants = cva(
  "inline-flex cursor-pointer items-center justify-center gap-2 whitespace-nowrap rounded-full text-sm font-bold transition-[background-color,color,box-shadow,transform] duration-200 active:scale-[0.98] active:shadow-pressed focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground shadow-raised-sm hover:bg-primary/90",
        // Health green: the main action on a screen (talk to a doctor,
        // request, accept).
        cta: "bg-cta text-cta-foreground shadow-raised-sm hover:bg-cta/90",
        destructive: "bg-destructive text-destructive-foreground hover:bg-destructive/90",
        outline:
          "border border-input bg-card text-foreground shadow-raised-sm hover:border-primary/50 hover:bg-accent",
        secondary: "bg-secondary text-secondary-foreground hover:bg-accent",
        ghost: "text-foreground/80 hover:bg-accent hover:text-foreground",
        link: "rounded-none text-primary-text underline-offset-4 hover:underline",
        // The emergency action. Its own variant rather than `destructive`,
        // so calling 102 can never look like a delete button.
        emergency:
          "bg-triage-red text-triage-red-foreground shadow-raised-sm hover:bg-triage-red/90",
      },
      size: {
        default: "h-11 px-5 sm:h-10",
        sm: "h-11 px-3.5 text-[13px] sm:h-9",
        lg: "h-12 px-7 text-base",
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
