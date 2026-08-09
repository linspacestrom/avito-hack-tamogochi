import ReactDOM from "react-dom/client";

import App from "./app/App";
import { AuthProvider } from "./providers/AuthProvider";

import "./index.css";

ReactDOM.createRoot(
  document.getElementById("root") as HTMLElement
).render(
  <AuthProvider>
    <App />
  </AuthProvider>
);