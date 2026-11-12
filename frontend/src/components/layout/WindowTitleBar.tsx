import type { CSSProperties, ReactNode } from "react";
import { cn } from "@opskat/ui";
import { useFullscreen } from "@/hooks/useFullscreen";
import { usePlatform } from "@/hooks/usePlatform";

export function WindowTitleBar({ title, children }: { title?: string; children?: ReactNode }) {
  const platform = usePlatform();
  const fullscreen = useFullscreen();
  return (
    <div
      data-testid="window-titlebar"
      className={cn(
        "flex h-10 w-full shrink-0 items-center gap-2 border-b border-panel-divider bg-sidebar",
        platform === "darwin" && !fullscreen ? "pl-20" : "pl-2",
        platform === "windows" ? "pr-[140px]" : "pr-2"
      )}
      style={{ "--wails-draggable": "drag" } as CSSProperties}
    >
      {title ? <span className="min-w-0 flex-1 truncate text-center text-sm font-medium">{title}</span> : children}
    </div>
  );
}
