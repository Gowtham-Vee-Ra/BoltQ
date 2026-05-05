import { useEffect, useRef, useCallback } from 'react';
import Pusher from 'pusher-js';

const PUSHER_KEY    = import.meta.env.VITE_PUSHER_KEY    || '41a375a1456772823568';
const PUSHER_CLUSTER = import.meta.env.VITE_PUSHER_CLUSTER || 'us2';
const CHANNEL       = 'boltq-notifications';
const EVENT         = 'new-notification';

export const usePusherNotifications = (onNotification) => {
  const pusherRef   = useRef(null);
  const channelRef  = useRef(null);
  const callbackRef = useRef(onNotification);

  // Keep callback ref fresh without re-subscribing
  useEffect(() => { callbackRef.current = onNotification; }, [onNotification]);

  useEffect(() => {
    pusherRef.current = new Pusher(PUSHER_KEY, { cluster: PUSHER_CLUSTER });
    channelRef.current = pusherRef.current.subscribe(CHANNEL);
    channelRef.current.bind(EVENT, (data) => callbackRef.current?.(data));

    return () => {
      channelRef.current?.unbind_all();
      pusherRef.current?.unsubscribe(CHANNEL);
      pusherRef.current?.disconnect();
    };
  }, []); // connect once, never re-run
};
