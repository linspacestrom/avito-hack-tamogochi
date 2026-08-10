import { createBrowserRouter, Navigate } from "react-router-dom";

import Pet from "../pages/Pet";
import Login from "../pages/Login";
import Register from "../pages/Register";
import ProtectedRoute from "../components/ProtectedRoute/ProtectedRoute";
import Rewards from "../pages/Rewards";


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
  {
    path: "/rewards",
    element: <Rewards />,
  },
]);

export default router;
