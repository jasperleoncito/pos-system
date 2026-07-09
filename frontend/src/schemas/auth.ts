import { z } from "zod";

const PASSWORD_MIN = 8;

export const loginSchema = z.object({
  email: z.string().email("Enter a valid email address"),
  password: z.string().min(1, "Password is required"),
});

export const registerSchema = z
  .object({
    full_name: z.string().min(2, "Name must be at least 2 characters").max(120),
    email: z.string().email("Enter a valid email address"),
    password: z.string().min(PASSWORD_MIN, `Password must be at least ${PASSWORD_MIN} characters`),
    confirm_password: z.string(),
    business_name: z.string().min(2, "Business name must be at least 2 characters").max(120),
    business_slug: z
      .string()
      .min(2, "URL slug must be at least 2 characters")
      .max(60)
      .regex(/^[a-z0-9]+(-[a-z0-9]+)*$/, "Use lowercase letters, numbers, and dashes"),
    plan: z.enum(["monthly", "yearly"]),
  })
  .refine((data) => data.password === data.confirm_password, {
    message: "Passwords do not match",
    path: ["confirm_password"],
  });

export const forgotPasswordSchema = z.object({
  email: z.string().email("Enter a valid email address"),
});

export const resetPasswordSchema = z
  .object({
    new_password: z.string().min(PASSWORD_MIN, `Password must be at least ${PASSWORD_MIN} characters`),
    confirm_password: z.string(),
  })
  .refine((data) => data.new_password === data.confirm_password, {
    message: "Passwords do not match",
    path: ["confirm_password"],
  });

export type LoginInput = z.infer<typeof loginSchema>;
export type RegisterInput = z.infer<typeof registerSchema>;
export type ForgotPasswordInput = z.infer<typeof forgotPasswordSchema>;
export type ResetPasswordInput = z.infer<typeof resetPasswordSchema>;
