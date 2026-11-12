import { useEffect } from "react";
import { EventsOn } from "@/lib/desktop";

/**
 * Subscribe to a Wails event with automatic cleanup. Unsubscribe with the
 * listener's own cancel: `EventsOff(name)` drops every listener for the name,
 * so it would silently detach another component's subscription to the same event.
 */
export function useWailsEvent<T = unknown>(eventName: string, handler: (data: T) => void) {
  useEffect(() => {
    const off = EventsOn(eventName, handler);
    return off;
  }, [eventName, handler]);
}
