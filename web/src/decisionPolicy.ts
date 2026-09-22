export type DecisionFacts = {
  operation:string; experimentId:string; conversationId:string;
  workspace:{id:string; revision:string}; experimentSessionId?:string;
  targetId:string; parametersDigest:string;
}

/** Decode authority-supplied scope; never derive it from a displayed command. */
export function decodeDecisionFacts(value:unknown):DecisionFacts {
  const record = (value:unknown):Record<string,unknown> => {
    if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('Invalid decision scope.')
    return value as Record<string,unknown>
  }
  const id = (value:unknown,empty=false):string => {
    if (empty && value === '') return ''
    if (typeof value !== 'string' || !/^[A-Za-z0-9][A-Za-z0-9._-]{0,95}$/.test(value)) throw new Error('Invalid decision identity.')
    return value
  }
  const source=record(value),workspace=record(source.workspace)
  if (typeof source.operation !== 'string' || !/^(native|gcs)\.[a-z.]{1,96}$/.test(source.operation)
    || typeof source.parametersDigest !== 'string' || !/^[a-f0-9]{64}$/.test(source.parametersDigest)
    || typeof workspace.revision !== 'string' || workspace.revision.length > 256 || /[\0\r\n]/.test(workspace.revision)) throw new Error('Invalid decision operation.')
  const native=source.operation.startsWith('native.')
  return {operation:source.operation,experimentId:id(source.experimentId),conversationId:id(source.conversationId,!native),
    workspace:{id:id(workspace.id,!native),revision:workspace.revision},targetId:id(source.targetId),parametersDigest:source.parametersDigest,
    ...(source.experimentSessionId === undefined ? {} : {experimentSessionId:id(source.experimentSessionId)})}
}
