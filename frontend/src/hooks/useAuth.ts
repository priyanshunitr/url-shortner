import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../services/api';
import { useRouter } from 'next/navigation';

export function useAuth() {
  const queryClient = useQueryClient();
  const router = useRouter();

  const loginMutation = useMutation({
    mutationFn: (credentials: Record<string, string>) => api.login(credentials),
    onSuccess: (data) => {
      if (typeof window !== 'undefined') {
        localStorage.setItem('token', data.token);
      }
      queryClient.setQueryData(['user'], data.user);
      router.push('/dashboard');
    },
  });

  const registerMutation = useMutation({
    mutationFn: (credentials: Record<string, string>) => api.register(credentials),
    onSuccess: (data) => {
      if (typeof window !== 'undefined') {
        localStorage.setItem('token', data.token);
      }
      queryClient.setQueryData(['user'], data.user);
      router.push('/dashboard');
    },
  });

  const logout = () => {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('token');
    }
    queryClient.setQueryData(['user'], null);
    router.push('/login');
  };

  return {
    login: loginMutation.mutateAsync,
    isLoggingIn: loginMutation.isPending,
    register: registerMutation.mutateAsync,
    isRegistering: registerMutation.isPending,
    logout,
  };
}
