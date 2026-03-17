export {};

declare global {
  interface Window {
    disco: {
      state: {
        view: string;
        [key: string]: unknown;
      };
      [key: string]: unknown;
    };
  }
}
