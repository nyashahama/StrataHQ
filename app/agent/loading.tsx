export default function Loading() {
  return (
    <div className="px-8 py-8 max-w-[900px]">
      <div className="h-4 w-32 bg-border rounded animate-pulse mb-6" />
      <div className="h-8 w-64 bg-border rounded animate-pulse mb-2" />
      <div className="h-4 w-80 bg-border rounded animate-pulse mb-8" />
      <div className="grid grid-cols-4 gap-4 mb-8">
        {[...Array(4)].map((_, i) => (
          <div key={i} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="h-7 w-16 bg-border rounded animate-pulse mb-2" />
            <div className="h-3 w-20 bg-border rounded animate-pulse" />
          </div>
        ))}
      </div>
      <div className="bg-white border border-border rounded-lg h-64 animate-pulse" />
    </div>
  )
}
