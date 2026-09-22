import { createContext, useContext } from 'react';
export const T3IdentityScope = createContext('');
export function useT3Identity() { return useContext(T3IdentityScope); }
