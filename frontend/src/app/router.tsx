import { createBrowserRouter, Navigate } from "react-router-dom";

import Pet from "../pages/Pet";
import Login from "../pages/Login";
import Register from "../pages/Register";
import ProtectedRoute from "../components/ProtectedRoute/ProtectedRoute";

const router = createBrowserRouter([
  {
    path: "/",
    element: <Navigate to="/login" replace />,
  },
  {
    path: "/login",
    element: <Login />,
  },
  {
    path: "/register",
    element: <Register />,
  },
  {
    path: "/pet",
    element: (
      <ProtectedRoute>
        <Pet />
      </ProtectedRoute>
    ),
  },
]);

export default router;
