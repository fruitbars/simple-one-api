/// <reference types="vite/client" />

interface Window {
  go?: {
    main?: {
      DesktopBridge?: {
        StreamChat(requestID: string, apiKey: string, payload: string): Promise<void>;
        CancelChat(requestID: string): Promise<void>;
      };
    };
  };
  runtime?: {
    EventsOn(eventName: string, callback: (...data: unknown[]) => void): () => void;
  };
}
