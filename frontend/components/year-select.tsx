"use client";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";

export function YearSelect({
  year,
  years,
  onChange,
  className,
  triggerClassName,
  size = "default",
}: {
  year: number;
  years: number[];
  onChange: (year: number) => void;
  className?: string;
  triggerClassName?: string;
  size?: "default" | "sm";
}) {
  const list = years.length ? years : [year];
  return (
    <div className={cn("space-y-1.5", className)}>
      <Select value={String(year)} onValueChange={(v) => onChange(Number(v))}>
        <SelectTrigger
          className={cn(
            size === "sm" && "h-8 text-xs",
            triggerClassName,
          )}
          aria-label="Select year"
        >
          <SelectValue placeholder="Year" />
        </SelectTrigger>
        <SelectContent>
          {list.map((y) => (
            <SelectItem key={y} value={String(y)}>
              {y}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
