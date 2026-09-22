import type { ToolData } from './upstream/t3/types.js'

export function AgentToolImages({ images, identity }: { images: NonNullable<ToolData['images']>; identity: string }) {
  return <div className="ms-7 mt-2 flex flex-col gap-2">
    {images.map((image, index) => <figure key={`${identity}:${index}`} className="m-0 min-w-0" data-xgc-role="agent-tool-image" data-xgc-id={`${identity}:${index}`}>
      <img src={image.src} alt={image.label} width={image.width} height={image.height}
        className="block h-auto max-h-64 max-w-full rounded-md object-contain object-left" loading="lazy" decoding="async"
        data-xgc-role="agent-tool-thumbnail" data-xgc-id={`${identity}:${index}`} />
      <figcaption className="mt-1 flex flex-wrap items-center gap-2 text-xs text-secondary-label">
        <span>{image.label}</span>
        {image.observedAt ? <time dateTime={image.observedAt} title={image.observedAt}>{new Date(image.observedAt).toLocaleTimeString()}</time> : null}
      </figcaption>
    </figure>)}
  </div>
}
